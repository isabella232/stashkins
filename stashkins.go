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
	stashURL                            = flag.String("stash-url", "http://stash.example.com", "Stash Base URL")
	jenkinsBaseURL                      = flag.String("jenkins-url", "http://jenkins.example.com", "Jenkins Base URL")
	listJobsWithoutFeatureBranches      = flag.Bool("jobs-without-feature-branches", false, "List jobs without feature branches")
	listJobsWithWildcardFeatureBranches = flag.Bool("jobs-with-wildcard-feature-branches", false, "List jobs with wildcard feature branches")
	listNonMavenJobs                    = flag.Bool("non-maven-jobs", false, "List non-maven jobs")
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

	if *listNonMavenJobs {
		jobs, err := jenkins.GetJobs(*jenkinsBaseURL)
		if err != nil {
			log.Fatalf("GetJobs Error: %v\n", err)
		}

		for _, v := range jobs {
			_, err := jenkins.GetJobConfig(*jenkinsBaseURL, v.Name)
			if err != nil {
				fmt.Printf("Non-maven2 job: %s\n", v.Name)
			}
		}
	}
}
