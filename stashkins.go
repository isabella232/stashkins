package main

import (
	"flag"
	"fmt"
	"log"
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
		log.Fatalf("Error: %v\n", err)
	}

	for _, v := range jobs {
		fmt.Printf("%+v\n", v)
	}
}
