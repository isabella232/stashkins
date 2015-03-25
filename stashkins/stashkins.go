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

	// Data structure for a Maven Jenkins Project
	MavenJob struct {
		JobName                    string // code in ssh://git@example.com:9999/teamp/code.git
		Description                string // mashup of repository URL and branch name
		BranchName                 string // feature/PROJ-999, as in feature/PROJ-999
		RepositoryURL              string // ssh://git@example.com:9999/teamp/code.git
		MavenSnapshotRepositoryURL string // the Maven repository URL to which to publish this job's artifacts
	}

	// Maps to a record in the template repository
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

		jobsWithGitURL   []jenkins.JobSummary
		branchesNotBuilt []string
		oldJobs          []jenkins.JobSummary

		StashClient   stash.Stash
		JenkinsClient jenkins.Jenkins
		NexusClient   maventools.Client

		StatelessOperations
	}

	StatelessOperations struct {
	}

	Aspect interface {
		MakeModel(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord Template) interface{}
		PostJobDeleteTasks(jobName, gitRepositoryURL, branchName string, templateRecord Template) error
		PostJobCreateTasks(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord Template) error
	}
)

var (
	Log *log.Logger = log.New(os.Stdout, "Stashkins ", log.Ldate|log.Ltime|log.Lshortfile)
)

func NewStashkins(stashParams, jenkinsParams WebClientParams, nexusParams MavenRepositoryParams) DefaultStashkins {
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
		StashParams:   stashParams,
		JenkinsParams: jenkinsParams,
		NexusParams:   nexusParams,
		StashClient:   stashClient,
		JenkinsClient: jenkinsClient,
		NexusClient:   nexusClient,
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
		Log.Printf("stashkins.ReconcileJobs get repository error: %v\n", err)
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
		if c.isTargetJob(jobSummary, gitRepository.SshUrl()) { // what if there is no ssh url?  only http?
			jobsWithGitURL = append(jobsWithGitURL, jobSummary)
		}
	}

	// Compile list of obsolete jobs
	oldJobs := make([]jenkins.JobSummary, 0)
	for _, jobSummary := range jobsWithGitURL {
		if c.shouldDeleteJob(jobSummary, stashBranches) {
			oldJobs = append(oldJobs, jobSummary)
		}
	}

	// Compile list of missing jobs
	branchesNotBuilt := make([]string, 0)
	for branch, _ := range stashBranches {
		if c.shouldCreateJob(jobSummaries, branch) {
			branchesNotBuilt = append(branchesNotBuilt, branch)
		}
	}

	fmt.Printf("jobsWithGitURL: %#v\n", jobsWithGitURL)
	fmt.Printf("oldJobs: %#v\n", oldJobs)
	fmt.Printf("branchesNotBuilt: %#v\n", branchesNotBuilt)

	/*
		// Delete old jobs
		for _, jobSummary := range oldJobs {
			jobName := jobSummary.JobDescriptor.Name
			if err := c.JenkinsClient.DeleteJob(jobName); err != nil {
				Log.Printf("stashkins.ReconcileJobs error deleting obsolete job %s, continuing:  %+v\n", jobName, err)
			} else {
				Log.Printf("Deleted obsolete job %+v\n", jobName)
			}

			jobAspect.PostJobDeleteTasks(jobName, gitRepository.SshUrl(), jobSummary.Branch, templateRecord)
		}

		// Create missing jobs
		for _, branch := range c.branchesNotBuilt {
			// For a branch feature/12, branchBaseName will be "feature" and branchSuffix will be "12".
			// For a branch named develop, branchBaseName will be develop and branchSuffix will be an empty string.
			branchBaseName, branchSuffix := c.suffixer(branch)

			newJobName := templateRecord.Slug + "-continuous-" + branchBaseName + branchSuffix
			newJobDescription := "This is a continuous build for " + templateRecord.Slug + ", branch " + branch

			model := jobAspect.MakeModel(newJobName, newJobDescription, gitRepository.SshUrl(), branch, templateRecord)

			if err := c.createJob(templateRecord, newJobName, model); err != nil {
				return err
			}

			jobAspect.PostJobCreateTasks(newJobName, newJobDescription, gitRepository.SshUrl(), branch, templateRecord)
		}
	*/
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

func (c StatelessOperations) suffixer(branch string) (string, string) {
	s := strings.Split(branch, "/")
	prefix := s[0]
	var suffix string

	if len(s) == 1 {
		return prefix, suffix
	}

	if len(s) == 2 {
		suffix = s[1]
	} else {
		suffix = branch[strings.Index(branch, "/")+1:]
		suffix = strings.Replace(suffix, "/", "-", -1)
	}
	return prefix, "-" + suffix
}

func (c StatelessOperations) branchIsManaged(stashBranch string) bool {
	return c.isFeatureBranch(stashBranch) || stashBranch == "develop"
}

func (c StatelessOperations) isFeatureBranch(branchName string) bool {
	// Do not try to manage a branch that has an * asterisk in it, as some Jenkins branch specs might contain (origin/feature/*).
	return strings.Contains(branchName, "feature/") && !strings.Contains(branchName, "*")
}

func (c StatelessOperations) isTargetJob(jobSummary jenkins.JobSummary, jobRepositoryURL string) bool {
	return jobSummary.GitURL == jobRepositoryURL
}

func (c StatelessOperations) shouldDeleteJob(jobSummary jenkins.JobSummary, stashBranches map[string]stash.Branch) bool {
	if !c.branchIsManaged(jobSummary.Branch) {
		return false
	}
	deleteJobConfig := true
	for stashBranch, _ := range stashBranches {
		if strings.HasSuffix(jobSummary.Branch, stashBranch) {
			deleteJobConfig = false
		}
	}
	return deleteJobConfig
}

func (c StatelessOperations) shouldCreateJob(jobSummaries []jenkins.JobSummary, branch string) bool {
	if !c.branchIsManaged(branch) {
		return false
	}
	for _, jobSummary := range jobSummaries {
		if strings.HasSuffix(jobSummary.Branch, branch) {
			return false
		}
	}
	return true
}
