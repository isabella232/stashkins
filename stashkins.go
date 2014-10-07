package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)
import "github.com/xoom/jenkins"

var (
	stashBaseURL   = flag.String("stash-url", "http://stash.example.com:8080", "Stash Base URL")
	jenkinsBaseURL = flag.String("jenkins-url", "http://jenkins.example.com:8080", "Jenkins Base URL")

	jobTemplateFile  = flag.String("job-template-file", "job-template.xml", "Jenkins job template file.")
	jobReport        = flag.Bool("job-report", false, "Show Jenkins/Stash sync state for job.  Requires -job-repository-url.")
	jobRepositoryURL = flag.String("job-repository-url", "ssh://git@example.com:9999/teamp/code.git", "The Git repository URL for this Jenkins job.")

	stashUserName = flag.String("Stash username", "", "Username for Stash authentication")
	stashPassword = flag.String("Stash password", "", "Password for Stash authentication")
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
		log.Printf("Analyzing %s...\n", *jobRepositoryURL)
		appConfigs := make([]jenkins.JobConfig, 0)
		for _, job := range allJobs {
			jobConfig, err := jenkins.GetJobConfig(*jenkinsBaseURL, job.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v, skipping...\n", job.Name, err)
			}
			for _, remoteCfg := range jobConfig.SCM.UserRemoteConfigs.UserRemoteConfig {
				fmt.Fprintf(os.Stderr, "checking url %s\n", remoteCfg.URL)
				if remoteCfg.URL == *jobRepositoryURL {
					appConfigs = append(appConfigs, jobConfig)
				}
			}
		}
		branches := make([]jenkins.Branch, 0)
		for _, v := range appConfigs {
			for _, branch := range v.SCM.Branches.Branch {
				branches = append(branches, branch)
			}
		}
		for _, v := range branches {
			fmt.Printf("%s\n", v.Name)
		}
	}
}
