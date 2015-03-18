package stashkins

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/xoom/jenkins"
	"github.com/xoom/maventools/nexus"
	"github.com/xoom/stash"
)

type (
	// Maps to a record in the template repository
	Template struct {
		ProjectKey  string
		Slug        string
		JobTemplate []byte
		JobType     jenkins.JobType
	}

	Stashkins interface {
		CreateNewJobs(Template) error
		DeleteOldJobs() error
		PostProcess() error
	}

	Common interface {
		suffixer(branch string) (string, string)
		branchIsManaged(stashBranch string) bool
		isFeatureBranch(branchName string) bool
		isTargetJob(jobSummary jenkins.JobSummary, jobRepositoryURL string) bool
		shouldDeleteJob(jobSummary jenkins.JobSummary, stashBranches map[string]stash.Branch) bool
		shouldCreateJob(jobSummaries []jenkins.JobSummary, branch string) bool
	}

	WebClientParams struct {
		URL      string
		UserName string
		Password string
	}

	ScmInfo struct {
		ProjectKey string
		Slug       string
	}

	DefaultStashkins struct {
		StashParams   WebClientParams
		JenkinsParams WebClientParams
		NexusParams   WebClientParams
		CommonOperations
		Stashkins

		jobsWithGitURL []jenkins.JobSummary
		missingJobs    []jenkins.JobSummary
		oldJobs        []jenkins.JobSummary
	}

	CommonOperations struct {
		stashClient   stash.Stash
		jenkinsClient jenkins.Jenkins
		nexusClient   nexus.Client
		Common
	}
)

func NewStashkins(stashParams, jenkinsParams, nexusParams WebClientParams) Stashkins {
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
		stashClient:    stashClient,
		jenkinsClient:  jenkinsClient,
		nexusClient:    nexusClient,
		jobsWithGitURL: make([]jenkins.JobSummary, 0),
		missingJobs:    make([]jenkins.JobSummary, 0),
		oldJobs:        make([]jenkins.JobSummary, 0),
	}
}

func (c DefaultStashkins) CreateNewJobs(template Template) error {
	repo, err := c.stashClient.GetRepository(template.ProjectKey, template.Slug)
	if err != nil {
		log.Fatalf("stashkins.main GetRepository error %v\n", err)
	}

	// Fetch all branches for this repository
	stashBranches, err := c.stashClient.GetBranches(template.ProjectKey, template.Slug)
	if err != nil {
		log.Printf("stashkins.CreateNewJobs error getting branches from Stash for repository %s/%s: %v\n", template.ProjectKey, template.Slug, err)
		return err
	}

	// Fetch job summaries
	// todo on jenkins client, create a jobs cache so this call can remain in the loop
	jobSummaries, err := c.jenkinsClient.GetJobSummaries()
	if err != nil {
		log.Printf("stashkins.CreateNewJobs get jobs error: %v\n", err)
		return err
	}

	gitRepository, err := c.stashClient.GetRepository(template.ProjectKey, template.Slug)
	if err != nil {
		log.Printf("stashkins.CreateNewJobs get jobs error: %v\n", err)
		return err
	}

	// Compile list of jobs that build anywhere on this Git repository
	for _, jobSummary := range jobSummaries {
		if c.isTargetJob(jobSummary, gitRepository.SshUrl()) {  // what if there is no ssh url?  only http?
			c.jobsWithGitURL = append(c.jobsWithGitURL, jobSummary)
		}
	}

	// Compile list of obsolete jobs
	for _, jobSummary := range c.jobsWithGitURL {
		if c.shouldDeleteJob(jobSummary, stashBranches) {
			c.oldJobs = append(c.oldJobs, jobSummary)
		}
	}

	// Compile list of missing jobs
    // todo doesn't really belong in a create-like method, but since we don't cache branches keep it for now
	for branch, _ := range stashBranches {
		if c.shouldCreateJob(jobSummaries, branch) {
			c.missingJobs = append(c.missingJobs, branch)
		}
	}



	return nil
}

func (c DefaultStashkins) DeleteOldJobs() error {
	return nil
}

func (c DefaultStashkins) PostProcess() error {
	return nil
}

func (c DefaultStashkins) getStashRepository() error {
	return nil
}

func (c CommonOperations) suffixer(branch string) (string, string) {
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

func (c CommonOperations) branchIsManaged(stashBranch string) bool {
	return c.isFeatureBranch(stashBranch) || stashBranch == "develop"
}

func (c CommonOperations) isFeatureBranch(branchName string) bool {
	// Do not try to manage a branch that has an * asterisk in it, as some Jenkins branch specs might contain (origin/feature/*).
	return strings.Contains(branchName, "feature/") && !strings.Contains(branchName, "*")
}

func (c CommonOperations) isTargetJob(jobSummary jenkins.JobSummary, jobRepositoryURL string) bool {
	return jobSummary.GitURL == jobRepositoryURL
}

func (c CommonOperations) shouldDeleteJob(jobSummary jenkins.JobSummary, stashBranches map[string]stash.Branch) bool {
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

func (c CommonOperations) shouldCreateJob(jobSummaries []jenkins.JobSummary, branch string) bool {
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
