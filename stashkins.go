package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
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

	listJobsWithoutFeatureBranches      = flag.Bool("jobs-without-feature-branches", false, "List jobs without feature branches")
	listJobsWithWildcardFeatureBranches = flag.Bool("jobs-with-wildcard-feature-branches", false, "List jobs with wildcard feature branches")
	listNonMavenJobs                    = flag.Bool("non-maven-jobs", false, "List non-maven jobs")
	listJobRepositories                 = flag.Bool("job-repositories", false, "List job repositories")
)

func init() {
	flag.Parse()
}

func main() {

	if *listJobsWithWildcardFeatureBranches {
		jobs, err := jenkins.GetJobs(*jenkinsBaseURL)
		if err != nil {
			log.Fatalf("GetJobs Error: %v\n", err)
		}

		for _, job := range jobs {
			jobConfig, err := jenkins.GetJobConfig(*jenkinsBaseURL, job.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v, skipping...\n", job.Name, err)
			}
			for _, branch := range jobConfig.SCM.Branches.Branch {
				if strings.Contains(branch.Name, "feature") && strings.Contains(branch.Name, "*") {
					fmt.Printf("%s has branch wildcards: %s\n", job.URL, branch.Name)
				}
			}

		}
	}
}
