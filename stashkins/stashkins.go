package stashkins

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"text/template"

	"github.com/xoom/jenkins"
	"github.com/xoom/maventools"
	"github.com/xoom/stash"
)

type (

	// Maven job model.  The name of these fields cannot be changed without
	// changing the same names in the text templates in the template repository.
	MavenJob struct {
		JobName                    string // code in ssh://git@example.com:9999/teamp/code.git
		Description                string // mashup of repository URL and branch name
		BranchName                 string // feature/PROJ-999, as in feature/PROJ-999
		RepositoryURL              string // ssh://git@example.com:9999/teamp/code.git
		MavenSnapshotRepositoryURL string // the Maven repository URL to which to publish this job's artifacts
		MavenRepositoryID          string // the unique id of the Maven repository to which this job's artifacts will be published
	}

	// Freestyle job model
	FreestyleJob struct {
		JobName       string // code in ssh://git@example.com:9999/teamp/code.git
		Description   string // mashup of repository URL and branch name
		BranchName    string // feature/PROJ-999, as in feature/PROJ-999
		RepositoryURL string // ssh://git@example.com:9999/teamp/code.git
	}

	// Generic struct to hold a network URL and login
	WebClientParams struct {
		URL      string
		UserName string
		Password string
	}

	// A Nexus / Maven client needs more than a URL and login, namely, a feature branch repository ID.
	MavenRepositoryParams struct {
		FeatureBranchRepositoryGroupID string
		WebClientParams
	}

	// The core Stashkins functionality is articulated here.
	DefaultStashkins struct {
		stashParams   WebClientParams
		jenkinsParams WebClientParams
		nexusParams   MavenRepositoryParams

		stashClient   stash.Stash
		jenkinsClient jenkins.Jenkins
		NexusClient   maventools.NexusClient

		branchOperations BranchOperations
	}

	// A record in the template repository
	JobTemplate struct {
		ProjectKey            string
		Slug                  string
		ContinuousJobTemplate []byte
		ReleaseJobTemplate    []byte
		JobType               jenkins.JobType
	}

	// Jobs have aspects.  Maven jobs create and delete per-branch repositories.
	Aspect interface {
		MakeModel(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord JobTemplate) interface{}
		PostJobDeleteTasks(jobName, gitRepositoryURL, branchName string, templateRecord JobTemplate) error
		PostJobCreateTasks(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord JobTemplate) error
	}
)

var (
	Log *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
)

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

	nexusClient := maventools.NewNexusClient(nexusParams.URL, nexusParams.UserName, nexusParams.Password)

	return DefaultStashkins{
		stashParams:      stashParams,
		jenkinsParams:    jenkinsParams,
		nexusParams:      nexusParams,
		stashClient:      stashClient,
		jenkinsClient:    jenkinsClient,
		branchOperations: branchOperations,
		NexusClient:      nexusClient,
	}
}

func (c DefaultStashkins) JobSummariesOverHTTP() ([]jenkins.JobSummary, error) {
	jobSummaries, err := c.jenkinsClient.GetJobSummaries()
	if err != nil {
		Log.Printf("stashkins.getJobSummaries get jobs error: %v\n", err)
		return nil, err
	}
	return jobSummaries, nil
}

func (c DefaultStashkins) JobSummariesFromFilesystem(root string) ([]jenkins.JobSummary, error) {
	jobSummaries, err := c.jenkinsClient.GetJobSummariesFromFilesystem(root)
	if err != nil {
		Log.Printf("stashkins.getJobSummariesFromFilesystem get jobs error: %v\n", err)
		return nil, err
	}
	return jobSummaries, nil
}

func (c DefaultStashkins) ReconcileJobs(jobSummaries []jenkins.JobSummary, jobTemplate JobTemplate, jobAspect Aspect) error {

	// Fetch the repository metadata
	gitRepository, err := c.stashClient.GetRepository(jobTemplate.ProjectKey, jobTemplate.Slug)
	if err != nil {
		Log.Printf("stashkins.ReconcileJobs get project repository error: %v\n", err)
		return err
	}

	// Fetch all branches for this repository
	stashBranches, err := c.stashClient.GetBranches(jobTemplate.ProjectKey, jobTemplate.Slug)
	if err != nil {
		Log.Printf("stashkins.ReconcileJobs error getting branches from Stash for repository %s/%s: %v\n", jobTemplate.ProjectKey, jobTemplate.Slug, err)
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

	Log.Printf("Number of Git branches for %s/%s: %d\n", jobTemplate.ProjectKey, jobTemplate.Slug, len(stashBranches))
	Log.Printf("Number of jobs building some branch against %s/%s: %d\n", jobTemplate.ProjectKey, jobTemplate.Slug, len(jobsWithGitURL))
	Log.Printf("Number of old jobs built against %s/%s: %d\n", jobTemplate.ProjectKey, jobTemplate.Slug, len(oldJobs))
	Log.Printf("Number of jobs to be created against %s/%s: %d\n", jobTemplate.ProjectKey, jobTemplate.Slug, len(branchesNotBuilt))

	// Delete old jobs
	for _, jobSummary := range oldJobs {
		jobName := jobSummary.JobDescriptor.Name
		if err := c.jenkinsClient.DeleteJob(jobName); err != nil {
			Log.Printf("stashkins.ReconcileJobs error deleting obsolete job %s, continuing:  %+v\n", jobName, err)
			continue
		} else {
			Log.Printf("Deleted obsolete job %+v\n", jobName)
		}

		if err := jobAspect.PostJobDeleteTasks(jobName, gitRepository.SshUrl(), jobSummary.Branch, jobTemplate); err != nil {
			Log.Printf("Error in post-job-delete-task, but willing to continue: %#v\n", err)
		}
	}

	// Create missing jobs
	for _, branch := range branchesNotBuilt {
		// For a branch feature/12, branchBaseName will be "feature" and branchSuffix will be "12".
		// For a branch named develop, branchBaseName will be develop and branchSuffix will be an empty string.
		branchBaseName, branchSuffix := c.branchOperations.suffixer(branch)

		newJobName := jobTemplate.ProjectKey + "-" + jobTemplate.Slug + "-continuous-" + branchBaseName + branchSuffix
		newJobDescription := "This is a continuous build for " + jobTemplate.ProjectKey + "-" + jobTemplate.Slug + ", branch " + branch

		model := jobAspect.MakeModel(newJobName, newJobDescription, gitRepository.SshUrl(), branch, jobTemplate)

		if err := c.createJob(jobTemplate.ContinuousJobTemplate, newJobName, model); err != nil {
			Log.Printf("Error creating continuous job %s:: %#v\n", newJobName, err)
			continue
		}

		if err := jobAspect.PostJobCreateTasks(newJobName, newJobDescription, gitRepository.SshUrl(), branch, jobTemplate); err != nil {
			Log.Printf("Error in post-job-create-task, but willing to continue: %#v\n", err)
		}
	}

	// Create release job.  The only time we can know if a release job should be built is when there are zero jobs building against this repository.
	// The reason is that there is no robust way to analyze an existing job building on origin/develop and know whether it is purposed for continuous or release.
	if len(jobsWithGitURL) == 0 {
		newJobName := jobTemplate.ProjectKey + "-" + jobTemplate.Slug + "-release"
		newJobDescription := "This is a release job for " + jobTemplate.ProjectKey + "-" + jobTemplate.Slug
		model := jobAspect.MakeModel(newJobName, newJobDescription, gitRepository.SshUrl(), "develop", jobTemplate)
		if err := c.createJob(jobTemplate.ReleaseJobTemplate, newJobName, model); err != nil {
			Log.Printf("Error creating release job %s: %#v\n", newJobName, err)
			return err
		}
	}

	return nil
}

func (c DefaultStashkins) createJob(data []byte, newJobName string, jobModel interface{}) error {
	if data == nil {
		Log.Printf("Template data is nil for job %s.  Skipping.\n", newJobName)
		return nil
	}

	jobTemplate, err := template.New("jobconfig").Parse(string(data))
	hydratedTemplate := bytes.NewBufferString("")
	err = jobTemplate.Execute(hydratedTemplate, jobModel)
	if err != nil {
		Log.Printf("stashkins.createJob cannot hydrate job template %s: %v\n", string(data), err)
		// If the template is bad, just return vs. continue because it won't work the next time through, either.
		return err
	}

	// Create the job
	err = c.jenkinsClient.CreateJob(newJobName, string(hydratedTemplate.Bytes()))
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
