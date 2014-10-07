package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)
import (
	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
)

var (
	stashBaseURL   = flag.String("stash-url", "http://stash.example.com:8080", "Stash Base URL")
	jenkinsBaseURL = flag.String("jenkins-url", "http://jenkins.example.com:8080", "Jenkins Base URL")

	jobTemplateFile  = flag.String("job-template-file", "job-template.xml", "Jenkins job template file.")
	jobReport        = flag.Bool("job-report", false, "Show Jenkins/Stash sync state for job.  Requires -job-repository-url.")
	jobRepositoryURL = flag.String("job-repository-url", "ssh://git@example.com:9999/teamp/code.git", "The Git repository URL for this Jenkins job.")

	stashUserName = flag.String("stash-username", "", "Username for Stash authentication")
	stashPassword = flag.String("stash-password", "", "Password for Stash authentication")
)

func init() {
	flag.Parse()
}

func main() {
	allJobs, err := jenkins.GetJobs(*jenkinsBaseURL)
	if err != nil {
		log.Fatalf("GetJobs Error: %v\n", err)
	}

	if *jobReport {
		fmt.Fprintf(os.Stderr, "Analyzing %s...\n", *jobRepositoryURL)

		// Jenkins jobs which build against a branch under the Git URL
		appJobConfigs := make([]jenkins.JobConfig, 0)
		for _, job := range allJobs {
			jobConfig, err := jenkins.GetJobConfig(*jenkinsBaseURL, job.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v, skipping...\n", job.Name, err)
			}
			for _, remoteCfg := range jobConfig.SCM.UserRemoteConfigs.UserRemoteConfig {
				if strings.HasPrefix(remoteCfg.URL, "http") {
					fmt.Fprintf(os.Stderr, "Found a job Git http URL.  This is not supported: %s\n", remoteCfg.URL)
				}
				fmt.Fprintf(os.Stderr, "checking url %s\n", remoteCfg.URL)
				if remoteCfg.URL == *jobRepositoryURL {
					appJobConfigs = append(appJobConfigs, jobConfig)
				}
			}
		}

		// Get Stash branches for this repository.
		repos, err := stash.GetRepositories(*stashBaseURL)
		if err != nil {
			log.Fatalf("Cannot get Stash repositories")
		}
		repo, ok := stash.HasRepository(repos, *jobRepositoryURL)
		if !ok {
			log.Fatalf("Unknown repository: %s\n", *jobRepositoryURL) // delete all entries of appJobConfigs - there is no repo here?
		}

		stashBranches, err := stash.GetBranches(*stashBaseURL, *stashUserName, *stashPassword, repo.Project.Key, repo.Slug)
		if err != nil {
			log.Fatalf("Cannot get Stash branches for repository: %s\n", *jobRepositoryURL)
		}

		fmt.Printf("Stash branches\n")
		for _, v := range stashBranches {
			fmt.Printf("	%s\n", v.DisplayID)
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
		fmt.Printf("Obsolete jobs: %+v\n", obsoleteJobs)
	}
}
