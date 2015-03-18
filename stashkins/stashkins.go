package stashkins

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
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

	Common interface {
		suffixer(branch string) (string, string)
		branchIsManaged(stashBranch string) bool
		isFeatureBranch(branchName string) bool
		isTargetJob(jobSummary jenkins.JobSummary, jobRepositoryURL string) bool
		shouldDeleteJob(jobSummary jenkins.JobSummary, stashBranches map[string]stash.Branch) bool
		shouldCreateJob(jobSummaries []jenkins.JobSummary, branch string) bool
	}

	MavenRepositoryParams struct {
		PerBranchRepositoryID string
		WebClientParams
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
		NexusParams   MavenRepositoryParams

		jobsWithGitURL   []jenkins.JobSummary
		branchesNotBuilt []string
		oldJobs          []jenkins.JobSummary

		CommonOperations
	}

	CommonOperations struct {
		stashClient   stash.Stash
		jenkinsClient jenkins.Jenkins
		nexusClient   nexus.Client
		Common
	}
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
		StashParams:      stashParams,
		JenkinsParams:    jenkinsParams,
		NexusParams:      nexusParams,
		CommonOperations: CommonOperations{stashClient: stashClient, jenkinsClient: jenkinsClient, nexusClient: nexusClient},
		jobsWithGitURL:   make([]jenkins.JobSummary, 0),
		branchesNotBuilt: make([]string, 0),
		oldJobs:          make([]jenkins.JobSummary, 0),
	}
}

func (c DefaultStashkins) Run(templateRecord Template) error {

	// Fetch all branches for this repository
	stashBranches, err := c.stashClient.GetBranches(templateRecord.ProjectKey, templateRecord.Slug)
	if err != nil {
		log.Printf("stashkins.CreateNewJobs error getting branches from Stash for repository %s/%s: %v\n", templateRecord.ProjectKey, templateRecord.Slug, err)
		return err
	}

	// Fetch job summaries
	// todo on jenkins client, create a jobs cache so this call can remain in the loop
	jobSummaries, err := c.jenkinsClient.GetJobSummaries()
	if err != nil {
		log.Printf("stashkins.CreateNewJobs get jobs error: %v\n", err)
		return err
	}

	gitRepository, err := c.stashClient.GetRepository(templateRecord.ProjectKey, templateRecord.Slug)
	if err != nil {
		log.Printf("stashkins.CreateNewJobs get jobs error: %v\n", err)
		return err
	}

	// Compile list of jobs that build anywhere on this Git repository
	for _, jobSummary := range jobSummaries {
		if c.isTargetJob(jobSummary, gitRepository.SshUrl()) { // what if there is no ssh url?  only http?
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
			c.branchesNotBuilt = append(c.branchesNotBuilt, branch)
		}
	}

	// Delete old jobs
	for _, jobSummary := range c.oldJobs {
		jobName := jobSummary.JobDescriptor.Name
		if err := c.jenkinsClient.DeleteJob(jobName); err != nil {
			log.Printf("stashkins.CreateNewJobs error deleting obsolete job %s, continuing:  %+v\n", jobName, err)
		} else {
			log.Printf("Deleted obsolete job %+v\n", jobName)
		}

		switch jobSummary.JobType {
		case jenkins.Maven:
			if c.isFeatureBranch(jobSummary.Branch) {
				var branchRepresentation string
				if strings.HasPrefix(jobSummary.Branch, "origin/") {
					branchRepresentation = jobSummary.Branch[len("origin/"):]
				}
				branchRepresentation = strings.Replace(branchRepresentation, "/", "_", -1)
				repositoryID := maventools.RepositoryID(fmt.Sprintf("%s.%s.%s", templateRecord.ProjectKey, templateRecord.Slug, branchRepresentation))
				if _, err := c.nexusClient.DeleteRepository(repositoryID); err != nil {
					log.Printf("stashkins.main failed to delete Maven repository %s: %+v\n", repositoryID, err)
				} else {
					log.Printf("Deleted Maven repository %v\n", repositoryID)
				}
			}
		}
	}

	// Create missing jobs
	for _, branch := range c.branchesNotBuilt {
		// For a branch feature/12, branchBaseName will be "feature" and branchSuffix will be "12".
		// For a branch named develop, branchBaseName will be develop and branchSuffix will be an empty string.
		branchBaseName, branchSuffix := c.suffixer(branch)

		newJobName := templateRecord.Slug + "-continuous-" + branchBaseName + branchSuffix
		newJobDescription := "This is a continuous build for " + templateRecord.Slug + ", branch " + branch

		switch templateRecord.JobType {
		case jenkins.Maven:
			mavenSnapshotRepositoryURL := c.buildMavenRepositoryURL(c.NexusParams.URL, templateRecord.ProjectKey, templateRecord.Slug, branch)

			jobDescr := MavenJob{
				JobName:                    newJobName,
				Description:                newJobDescription,
				BranchName:                 branch,
				RepositoryURL:              gitRepository.SshUrl(),
				MavenSnapshotRepositoryURL: mavenSnapshotRepositoryURL,
			}

			// Prepare the job template
			jobTemplate, err := template.New("jobconfig").Parse(string(templateRecord.JobTemplate))

			hydratedTemplate := bytes.NewBufferString("")
			err = jobTemplate.Execute(hydratedTemplate, jobDescr)
			if err != nil {
				log.Printf("stashkins.XXX cannot hydrate job template %s: %v\n", string(templateRecord.JobTemplate), err)
				// If the template is bad, just return vs. continue because it won't work the next time through, either.
				return err
			}

			// Create the job
			err = c.jenkinsClient.CreateJob(newJobName, string(hydratedTemplate.Bytes()))
			if err != nil {
				log.Printf("stashkins.XXX failed to create job %+v, continuing...: error==%#v\n", jobDescr, err)
			} else {
				log.Printf("Created job %s\n", newJobName)
			}

			// Feature branches get a dedicated per-branch Nexus Maven repository
			if c.isFeatureBranch(branch) {
				branchRepresentation := strings.Replace(branch, "/", "_", -1)
				repositoryID := maventools.RepositoryID(fmt.Sprintf("%s.%s.%s", templateRecord.ProjectKey, templateRecord.Slug, branchRepresentation))
				if present, err := c.nexusClient.RepositoryExists(repositoryID); err == nil && !present {
					if rc, err := c.nexusClient.CreateSnapshotRepository(repositoryID); err != nil {
						log.Printf("stashkins.main failed to create Maven repository %s: %+v\n", repositoryID, err)
					} else {
						if rc == 201 {
							log.Printf("Created Maven repositoryID %s\n", repositoryID)
						}
					}
				} else {
					if err != nil {
						log.Printf("stashkins.main error creating Maven repositoryID %s: %v\n", repositoryID, err)
					} else {
						log.Printf("stashkins.main Maven repositoryID %s exists.  Skipping.\n", repositoryID)
					}
				}
				repositoryGroupID := maventools.GroupID(c.NexusParams.PerBranchRepositoryID)
				if rc, err := c.nexusClient.AddRepositoryToGroup(repositoryID, repositoryGroupID); err != nil {
					log.Printf("stashkins.main failed to add Maven repository %s to repository group %s: %+v\n", repositoryID, c.NexusParams.PerBranchRepositoryID, err)
				} else {
					if rc == 200 {
						log.Printf("Maven repositoryID %s added to repository groupID %s\n", repositoryID, c.NexusParams.PerBranchRepositoryID)
					}
				}
			}
		}
	}

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

// Form the maven repository ID from project parts.  Each part must be cleaned and made URL-safe because the result will form part of an HTTP URL.
func (c CommonOperations) mavenRepositoryID(gitRepoProjectKey, gitRepoSlug, gitBranch string) string {
	return fmt.Sprintf("%s.%s.%s", c.mavenRepoIDPartCleaner(gitRepoProjectKey), c.mavenRepoIDPartCleaner(gitRepoSlug), c.mavenRepoIDPartCleaner(gitBranch))
}

func (c CommonOperations) mavenRepoIDPartCleaner(b string) string {
	thing := b
	thing = strings.Replace(thing, "/", "_", -1)
	thing = strings.Replace(thing, "&", "_", -1)
	thing = strings.Replace(thing, "?", "_", -1)
	return thing
}

func (c CommonOperations) buildMavenRepositoryURL(nexusBaseURL, gitProjectKey, gitRepositorySlug, gitBranch string) string {
	var mavenSnapshotRepositoryURL string
	if gitBranch == "develop" {
		mavenSnapshotRepositoryURL = fmt.Sprintf("%s/content/repositories/snapshots", nexusBaseURL)
	} else {
		// For feature/ branches, use per-branch repositories
		mavenSnapshotRepositoryID := c.mavenRepositoryID(gitProjectKey, gitRepositorySlug, gitBranch)
		mavenSnapshotRepositoryURL = fmt.Sprintf("%s/content/repositories/%s", nexusBaseURL, mavenSnapshotRepositoryID)
	}
	return mavenSnapshotRepositoryURL
}
