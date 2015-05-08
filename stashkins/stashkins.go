package stashkins

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/xoom/jenkins"
	"github.com/xoom/maventools"
	"github.com/xoom/maventools/nexus"
	"github.com/xoom/stash"
)

type (

	// Go text/template data structure for a Maven Jenkins Project
	MavenJob struct {
		JobName                    string // code in ssh://git@example.com:9999/teamp/code.git
		Description                string // mashup of repository URL and branch name
		BranchName                 string // feature/PROJ-999, as in feature/PROJ-999
		RepositoryURL              string // ssh://git@example.com:9999/teamp/code.git
		MavenSnapshotRepositoryURL string // the Maven repository URL to which to publish this job's artifacts
	}

	// A record in the template repository
	Template struct {
		ProjectKey  string
		Slug        string
		JobTemplate []byte
		JobType     jenkins.JobType
	}

	WebClientParams struct {
		URL      string
		UserName string
		Password string
	}

	MavenRepositoryParams struct {
		PerBranchRepositoryID string
		WebClientParams
	}

	DefaultStashkins struct {
		StashParams   WebClientParams
		JenkinsParams WebClientParams
		NexusParams   MavenRepositoryParams

		StashClient   stash.Stash
		JenkinsClient jenkins.Jenkins
		NexusClient   maventools.Client

		branchOperations BranchOperations
	}

	Aspect interface {
		MakeModel(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord Template) interface{}
		PostJobDeleteTasks(jobName, gitRepositoryURL, branchName string, templateRecord Template) error
		PostJobCreateTasks(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord Template) error
	}
)

var (
	Log *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
)

func NewBranchOperations(managedPrefixes string) BranchOperations {
	t := strings.Split(managedPrefixes, ",")
	prefixes := make([]string, 0)
	for _, v := range t {
		candidate := strings.TrimSpace(v)
		if candidate == "" {
			Log.Printf("Skipping zero length managed prefix candidate\n.")
			continue
		}
		if !strings.HasSuffix(candidate, "/") {
			Log.Printf("Candidate missing trailing /.  Skipping.")
			continue
		}
		prefixes = append(prefixes, candidate)
	}
	return BranchOperations{ManagedPrefixes: prefixes}
}

func NewStashkins(stashParams, jenkinsParams WebClientParams, nexusParams MavenRepositoryParams, branchOperations BranchOperations) DefaultStashkins {
	var err error
	var stashURL *url.URL
	var jenkinsURL *url.URL

	stashURL, err = url.Parse(stashParams.URL)
	if err != nil {
		panic(fmt.Sprintf("Error parsing Stash URL %s: %v\n", stashParams.URL, err))
	}
	stashClient := stash.NewClient(stashParams.UserName, stashParams.Password, stashURL)

	jenkinsURL, err = url.Parse(jenkinsParams.URL)
	if err != nil {
		panic(fmt.Sprintf("Error parsing Jenkins URL %s: %v\n", jenkinsParams.URL, err))
	}
	jenkinsClient := jenkins.NewClient(jenkinsURL, jenkinsParams.UserName, jenkinsParams.Password)

	nexusClient := nexus.NewClient(nexusParams.URL, nexusParams.UserName, nexusParams.Password)

	return DefaultStashkins{
		StashParams:      stashParams,
		JenkinsParams:    jenkinsParams,
		NexusParams:      nexusParams,
		StashClient:      stashClient,
		JenkinsClient:    jenkinsClient,
		NexusClient:      nexusClient,
		branchOperations: branchOperations,
	}
}

func (c DefaultStashkins) GetJobSummaries() ([]jenkins.JobSummary, error) {
	jobSummaries, err := c.JenkinsClient.GetJobSummaries()
	if err != nil {
		Log.Printf("stashkins.getJobSummaries get jobs error: %v\n", err)
		return nil, err
	}
	return jobSummaries, nil
}

func (c DefaultStashkins) ReconcileJobs(jobSummaries []jenkins.JobSummary, templateRecord Template, jobAspect Aspect) error {

	// Fetch the repository metadata
	gitRepository, err := c.StashClient.GetRepository(templateRecord.ProjectKey, templateRecord.Slug)
	if err != nil {
		Log.Printf("stashkins.ReconcileJobs get project repository error: %v\n", err)
		return err
	}

	// Fetch all branches for this repository
	stashBranches, err := c.StashClient.GetBranches(templateRecord.ProjectKey, templateRecord.Slug)
	if err != nil {
		Log.Printf("stashkins.ReconcileJobs error getting branches from Stash for repository %s/%s: %v\n", templateRecord.ProjectKey, templateRecord.Slug, err)
		return err
	}

	// Compile list of jobs that build anywhere on this Git repository
	jobsWithGitURL := make([]jenkins.JobSummary, 0)
	for _, jobSummary := range jobSummaries {
		if c.isTargetJob(jobSummary, gitRepository.SshUrl()) {
			jobsWithGitURL = append(jobsWithGitURL, jobSummary)
		}
	}

	// Compile list of obsolete jobs
	oldJobs := make([]jenkins.JobSummary, 0)
	for _, jobSummary := range jobsWithGitURL {
		if c.branchOperations.shouldDeleteJob(jobSummary, stashBranches) {
			oldJobs = append(oldJobs, jobSummary)
		}
	}

	// Compile list of missing jobs
	branchesNotBuilt := make([]string, 0)
	for branch, _ := range stashBranches {
		if c.branchOperations.shouldCreateJob(jobsWithGitURL, branch) {
			branchesNotBuilt = append(branchesNotBuilt, branch)
		}
	}

	Log.Printf("Number of Git branches for %s/%s: %d\n", templateRecord.ProjectKey, templateRecord.Slug, len(stashBranches))
	Log.Printf("Number of jobs building some branch against %s/%s: %d\n", templateRecord.ProjectKey, templateRecord.Slug, len(jobsWithGitURL))
	Log.Printf("Number of old jobs built against %s/%s: %d\n", templateRecord.ProjectKey, templateRecord.Slug, len(oldJobs))
	Log.Printf("Number of jobs to be created against %s/%s: %d\n", templateRecord.ProjectKey, templateRecord.Slug, len(branchesNotBuilt))

	// Delete old jobs
	for _, jobSummary := range oldJobs {
		jobName := jobSummary.JobDescriptor.Name
		if err := c.JenkinsClient.DeleteJob(jobName); err != nil {
			Log.Printf("stashkins.ReconcileJobs error deleting obsolete job %s, continuing:  %+v\n", jobName, err)
			continue
		} else {
			Log.Printf("Deleted obsolete job %+v\n", jobName)
		}

		if err := jobAspect.PostJobDeleteTasks(jobName, gitRepository.SshUrl(), jobSummary.Branch, templateRecord); err != nil {
			Log.Printf("Error in post-job-delete-task, but willing to continue: %#v\n", err)
		}
	}

	// Create missing jobs
	for _, branch := range branchesNotBuilt {
		// For a branch feature/12, branchBaseName will be "feature" and branchSuffix will be "12".
		// For a branch named develop, branchBaseName will be develop and branchSuffix will be an empty string.
		branchBaseName, branchSuffix := c.branchOperations.suffixer(branch)

		newJobName := templateRecord.ProjectKey + "-" + templateRecord.Slug + "-continuous-" + branchBaseName + branchSuffix
		newJobDescription := "This is a continuous build for " + templateRecord.ProjectKey + "-" + templateRecord.Slug + ", branch " + branch

		model := jobAspect.MakeModel(newJobName, newJobDescription, gitRepository.SshUrl(), branch, templateRecord)

		if err := c.createJob(templateRecord, newJobName, model); err != nil {
			Log.Printf("Error creating job %s:: %#v\n", newJobName, err)
			continue
		}

		if err := jobAspect.PostJobCreateTasks(newJobName, newJobDescription, gitRepository.SshUrl(), branch, templateRecord); err != nil {
			Log.Printf("Error in post-job-create-task, but willing to continue: %#v\n", err)
		}
	}
	return nil
}

func (c DefaultStashkins) createJob(templateRecord Template, newJobName string, jobModel interface{}) error {
	jobTemplate, err := template.New("jobconfig").Parse(string(templateRecord.JobTemplate))
	hydratedTemplate := bytes.NewBufferString("")
	err = jobTemplate.Execute(hydratedTemplate, jobModel)
	if err != nil {
		Log.Printf("stashkins.createJob cannot hydrate job template %s: %v\n", string(templateRecord.JobTemplate), err)
		// If the template is bad, just return vs. continue because it won't work the next time through, either.
		return err
	}

	// Create the job
	err = c.JenkinsClient.CreateJob(newJobName, string(hydratedTemplate.Bytes()))
	if err != nil {
		Log.Printf("stashkins.createJob failed to create job %+v, continuing...: error==%#v\n", newJobName, err)
		return err
	} else {
		Log.Printf("Created job %s\n", newJobName)
	}

	return nil
}


func (c DefaultStashkins) isTargetJob(jobSummary jenkins.JobSummary, jobRepositoryURL string) bool {
	return jobSummary.GitURL == jobRepositoryURL
}


