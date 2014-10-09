package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
)

// JobTemplate is used to populate a template XML Jenkins job config file with appropriate values for prospective new jobs
type JobTemplate struct {
	JobName             string // code in ssh://git@example.com:9999/teamp/code.git
	Description         string // mashup of repository URL and branch name
	BranchName          string // feature/PROJ-999, as in feature/PROJ-999
	RepositoryURL       string // ssh://git@example.com:9999/teamp/code.git
	NexusRepositoryType string // if branch == master then releases else snapshots
}

var (
	stashBaseURL   = flag.String("stash-rest-base-url", "http://stash.example.com:8080", "Stash REST Base URL")
	jenkinsBaseURL = flag.String("jenkins-url", "http://jenkins.example.com:8080", "Jenkins Base URL")

	jobTemplateFile  = flag.String("job-template-file", "job-template.xml", "Jenkins job template file.")
	jobReport        = flag.Bool("job-report", false, "Show Jenkins/Stash sync state for job.  Requires -job-repository-url.")
	jobRepositoryURL = flag.String("job-repository-url", "ssh://git@example.com:9999/teamp/code.git", "The Git repository URL referenced by the Jenkins jobs.")

	stashUserName = flag.String("stash-username", "", "Username for Stash authentication")
	stashPassword = flag.String("stash-password", "", "Password for Stash authentication")
)

func init() {
	flag.Parse()
}

func main() {
	if *jobReport {
		// Get Stash repositoies.
		repos, err := stash.GetRepositories(*stashBaseURL)
		if err != nil {
			log.Fatalf("Cannot get Stash repositories: %v\n", err)
		}
		repo, ok := stash.HasRepository(repos, *jobRepositoryURL)
		if !ok {
			log.Fatalf("Repository not found in Stash: %s\n", *jobRepositoryURL)
		}

		fmt.Fprintf(os.Stderr, "Analyzing %s...\n", *jobRepositoryURL)

		allJobs, err := jenkins.GetJobs(*jenkinsBaseURL)
		if err != nil {
			log.Fatalf("GetJobs Error: %v\n", err)
		}

		// Jenkins jobs which build against a branch under the Git URL
		appJobConfigs := make([]jenkins.JobConfig, 0)
		for _, job := range allJobs {
			jobConfig, err := jenkins.GetJobConfig(*jenkinsBaseURL, job.Name)
			if err != nil {
				//fmt.Fprintf(os.Stderr, "%s: %v, skipping...\n", job.Name, err)
			}
			for _, remoteCfg := range jobConfig.SCM.UserRemoteConfigs.UserRemoteConfig {
				if strings.HasPrefix(remoteCfg.URL, "http") {
					fmt.Fprintf(os.Stderr, "Found a job Git http URL.  This is not supported: %s\n", remoteCfg.URL)
				}
				if remoteCfg.URL == *jobRepositoryURL {
					appJobConfigs = append(appJobConfigs, jobConfig)
				}
			}
		}

		stashBranches, err := stash.GetBranches(*stashBaseURL, *stashUserName, *stashPassword, repo.Project.Key, repo.Slug)
		if err != nil {
			log.Fatalf("Cannot get Stash branches for repository: %s\n", *jobRepositoryURL)
		}

		// Find branches Jenkins is building that no longer exist in Stash
		obsoleteJobs := make([]jenkins.JobConfig, 0)
		for _, jobConfig := range appJobConfigs {
			deleteJobConfig := true
			for _, builtBranch := range jobConfig.SCM.Branches.Branch {
				for stashBranch, _ := range stashBranches {
					if strings.HasSuffix(builtBranch.Name, stashBranch) {
						deleteJobConfig = false
					}
				}
			}
			if deleteJobConfig {
				obsoleteJobs = append(obsoleteJobs, jobConfig)
			}
		}
		if len(obsoleteJobs) > 0 {
			fmt.Fprintf(os.Stderr, "Number of obsolete jobs: %d\n", len(obsoleteJobs))
			for _, job := range obsoleteJobs {
				if err := jenkins.DeleteJob(*jenkinsBaseURL, job.JobName); err != nil {
					fmt.Fprintf(os.Stderr, "Error deleting obsolete job %s, continuing:  %+v\n", job.JobName, err)
				} else {
					fmt.Fprintf(os.Stderr, "Deleting obsolete job %+v\n", job.JobName)
				}
				// todo remove this when we want to delete more than just one
				break
			}
		}

		// Find missing jobs.  This is characterized as a branch in Stash that is not built by any job.
		missingJobs := make([]string, 0)
		for branch, _ := range stashBranches {
			missingJob := true
			for _, jobConfig := range appJobConfigs {
				for _, builtBranch := range jobConfig.SCM.Branches.Branch {
					if strings.HasSuffix(builtBranch.Name, branch) {
						missingJob = false
					}
				}
			}
			if missingJob {
				missingJobs = append(missingJobs, branch)
			}
		}
		if len(missingJobs) > 0 {
			fmt.Fprintf(os.Stderr, "Number of missing jobs: %d\n", len(missingJobs))

			// Create Jenkins jobs
			for _, branch := range missingJobs {
				var nexusType string
				if branch == "master" {
					nexusType = "releases"
				} else {
					nexusType = "snapshots"
				}

				var branchType string
				var branchSuffix string
				if branch == "master" || branch == "develop" || !strings.Contains(branch, "/") {
					branchType = branch
					branchSuffix = ""
				} else {
					branchType, branchSuffix = suffixer(branch)
				}

				jobDescr := JobTemplate{
					JobName:             repo.Slug + "-continuous-" + branchType + branchSuffix,
					Description:         "This is a continuous build for " + repo.Slug + ", branch " + branch,
					BranchName:          branch,
					RepositoryURL:       *jobRepositoryURL,
					NexusRepositoryType: nexusType,
				}

				// Prepare the job template
				data, err := ioutil.ReadFile(*jobTemplateFile)
				if err != nil {
					log.Fatalf("Cannot read job template file %s: %v\n", *jobTemplateFile, err)
				}
				jobTemplate, err := template.New("jobconfig").Parse(string(data))
				if err != nil {
					log.Fatalf("Cannot parse job template file %s: %v\n", *jobTemplateFile, err)
				}
				result := bytes.NewBufferString("")
				err = jobTemplate.Execute(result, jobDescr)
				if err != nil {
					log.Fatalf("Cannot execute job template file %s: %v\n", *jobTemplateFile, err)
				}
				templateString := string(result.Bytes())

				// Create the job
				err = jenkins.CreateJob(*jenkinsBaseURL, jobDescr.JobName, templateString)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to create job %+v, continuing...: error==%+v\n", jobDescr, err)
				}
				fmt.Fprintf(os.Stderr, "\n	created job %+v\n", jobDescr)

				// todo remove this when we want to do more than one
				os.Exit(0)
			}
		}
	}
}

func suffixer(branch string) (string, string) {
	s := strings.Split(branch, "/")
	prefix := s[0]
	var suffix string
	if len(s) == 2 {
		suffix = s[1]
	} else {
		suffix = branch[strings.Index(branch, "/")+1:]
		suffix = strings.Replace(suffix, "/", "-", -1)
	}
	return prefix, "-" + suffix
}
