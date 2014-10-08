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

type JobTemplate struct {
	AppName             string // code.git as in ssh://git@example.com:9999/teamp/code.git
	BranchType          string // feature, as in feature/PLAT-999
	BranchSuffix        string // PLAT-999 as in feature/PLAT-999
	RepositoryURL       string // ssh://git@example.com:9999/teamp/code.git
	NexusRepositoryType string // if branch == master then releases else snapshots
}

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

		// Get Stash branches for this repository.
		repos, err := stash.GetRepositories(*stashBaseURL)
		if err != nil {
			log.Fatalf("Cannot get Stash repositories: %v\n", err)
		}
		repo, ok := stash.HasRepository(repos, *jobRepositoryURL)
		if !ok {
			log.Fatalf("Unknown repository: %s\n", *jobRepositoryURL) // delete all entries of appJobConfigs - there is no repo here?
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
			fmt.Printf("Obsolete jobs\n", obsoleteJobs)
			for _, job := range obsoleteJobs {
				fmt.Printf("	%+v\n", job)
			}
		}

		// Find missing jobs
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
			fmt.Printf("Missing jobs\n")

			// Create Jenkins jobs
			for _, v := range missingJobs {
				appName := nameFromGitURL(*jobRepositoryURL)

				var nexusType string
				if v == "master" {
					nexusType = "releases"
				} else {
					nexusType = "snapshots"
				}

				var branchType string
				var branchSuffix string
				if v == "master" || v == "develop" || !strings.Contains(v, "/") {
					branchType = v
					branchSuffix = ""
				} else {
					branchType = strings.Split(v, "/")[0]
					branchSuffix = strings.Split(v, "/")[1]
				}

				jobDescr := JobTemplate{
					AppName:             appName,
					BranchType:          branchType,
					BranchSuffix:        branchSuffix,
					RepositoryURL:       *jobRepositoryURL,
					NexusRepositoryType: nexusType,
				}

				data, err := ioutil.ReadFile(*jobTemplateFile)
				if err != nil {
					log.Fatalf("Cannot read job template file %s: %v\n", *jobTemplateFile, err)
				}
				tmpl, err := template.New("jobconfig").Parse(string(data))
				if err != nil {
					log.Fatalf("Cannot parse job template file %s: %v\n", *jobTemplateFile, err)
				}
				result := bytes.NewBufferString("")
				err = tmpl.Execute(result, jobDescr)
				if err != nil {
					log.Fatalf("Cannot execute job template file %s: %v\n", *jobTemplateFile, err)
				}
				templ := string(result.Bytes())
				err = jenkins.CreateJob(*jenkinsBaseURL, appName, templ)
				if err != nil {
					fmt.Printf("Failed to create job %+v: %+v\n", jobDescr, err)
				}
				fmt.Printf("created job %+v\n", jobDescr)

				// just do one.
				os.Exit(0)
				/*
				 */
			}
		}
	}
}

func nameFromGitURL(url string) string {
	i := strings.LastIndex(url, "/") + 1
	return url[i:]
}
