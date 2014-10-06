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
	stashURL                       = flag.String("stash-url", "http://stash.example.com", "Stash Base URL")
	jenkinsBaseURL                 = flag.String("jenkins-url", "http://jenkins.example.com", "Jenkins Base URL")
	listJobsWithoutFeatureBranches = flag.Bool("jobs-without-feature-branches", true, "List jobs without feature branches")
)

func init() {
	flag.Parse()
}

func main() {
	jobs, err := jenkins.GetJobs(*jenkinsBaseURL)
	if err != nil {
		log.Fatalf("GetJobs Error: %v\n", err)
	}

	for _, v := range jobs {
		//fmt.Printf("%+v\n", v)
		jobConfig, err := jenkins.GetJobConfig(*jenkinsBaseURL, v.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v, skipping...\n", v.Name, err)
		}
		for _, branch := range jobConfig.SCM.Branches.Branch {
			if strings.Contains(branch.Name, "feature") && strings.Contains(branch.Name, "*") {
				fmt.Printf("%s has branch wildcards: %s\n", v.URL, branch.Name)
			}
		}

	}
}
