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
	"github.com/xoom/maventools/nexus"
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
	jobSync          = flag.Bool("job-sync", false, "Sync Jenkins state against Stash for a given Stash repository.  Requires -job-repository-url.")
	jobRepositoryURL = flag.String("job-repository-url", "ssh://git@example.com:9999/teamp/code.git", "The Git repository URL referenced by the Jenkins jobs.")

	stashUserName = flag.String("stash-username", "", "Username for Stash authentication")
	stashPassword = flag.String("stash-password", "", "Password for Stash authentication")

	mavenBaseURL           = flag.String("maven-repo-base-url", "http://localhost:8081/nexus", "Maven repository management Base URL")
	mavenUsername          = flag.String("maven-repo-username", "", "Username for Maven repository management")
	mavenPassword          = flag.String("maven-repo-password", "", "Password for Maven repository management")
	mavenRepositoryGroupID = flag.String("maven-repo-repository-groupID", "", "Repository groupID in which to create new per-branch repositories")
	doMavenRepoManagement  = flag.Bool("do-maven-repo-management", false, "Whether to create, delete and update Maven repositories as per-branch operations")

	versionFlag = flag.Bool("version", false, "Print build info from which stashkins was built")

	version   string
	commit    string
	buildTime string
	sdkInfo   string
)

func init() {
	flag.Parse()
}

func main() {
	log.Printf("Version: %s, CommitID: %s, build time: %s, SDK Info: %s\n", version, commit, buildTime, sdkInfo)
	if *versionFlag {
		os.Exit(0)
	}
	if *jobSync {

		if *doMavenRepoManagement && (*mavenUsername == "" || *mavenPassword == "" || *mavenRepositoryGroupID == "") {
			log.Fatalf("Nexus username, password, and repository group are required\n")
		}

		mavenRepositoryClient := nexus.NewClient(*mavenBaseURL, *mavenUsername, *mavenPassword)

		// Get Stash repositories.
		repos, err := stash.GetRepositories(*stashBaseURL)
		if err != nil {
			log.Fatalf("stash.GetRepositories error: %v\n", err)
		}
		repo, ok := stash.HasRepository(repos, *jobRepositoryURL)
		if !ok {
			log.Fatalf("stashkins.main repository not found in Stash: %s\n", *jobRepositoryURL)
		}

		log.Printf("Analyzing repository %s...\n", *jobRepositoryURL)

		allJobs, err := jenkins.GetJobs(*jenkinsBaseURL)
		if err != nil {
			log.Fatalf("stashkins.main get jobs error: %v\n", err)
		}

		// Jenkins jobs which build against a branch under the Git URL
		appJobConfigs := make([]jenkins.JobConfig, 0)
		for _, job := range allJobs {
			jobConfig, err := jenkins.GetJobConfig(*jenkinsBaseURL, job.Name)
			if err != nil {
				// This probably means the job config did not conform to the backing XML model we used.  Not a maven job.
				log.Printf("stashkins.main Jenkins GetJobConfig error for job %s: %v, skipping...\n", job.Name, err)
			}
			for _, remoteCfg := range jobConfig.SCM.UserRemoteConfigs.UserRemoteConfig {
				if strings.HasPrefix(remoteCfg.URL, "http") {
					log.Printf("Found a job Git http URL.  This is not supported: %s\n", remoteCfg.URL)
				}
				if remoteCfg.URL == *jobRepositoryURL {
					appJobConfigs = append(appJobConfigs, jobConfig)
				}
			}
		}

		stashBranches, err := stash.GetBranches(*stashBaseURL, *stashUserName, *stashPassword, repo.Project.Key, repo.Slug)
		if err != nil {
			log.Fatalf("stashkins.main error getting branches from Stash for repository %s: %v\n", *jobRepositoryURL, err)
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
			log.Printf("Number of obsolete jobs: %d\n", len(obsoleteJobs))
			for _, job := range obsoleteJobs {
				if err := jenkins.DeleteJob(*jenkinsBaseURL, job.JobName); err != nil {
					log.Printf("stashkins.main error deleting obsolete job %s, continuing:  %+v\n", job.JobName, err)
				} else {
					log.Printf("Deleting obsolete job %+v\n", job.JobName)
				}

				// Nexus
				if *doMavenRepoManagement {
					for _, branch := range job.SCM.Branches.Branch {
						branchRepresentation := strings.Replace(branch.Name, "origin/", "", 1)
						branchRepresentation = strings.Replace(branchRepresentation, "/", "_", -1)
						repositoryID := fmt.Sprintf("%s.%s.%s", repo.Project.Key, repo.Slug, branchRepresentation)
						if err := mavenRepositoryClient.DeleteRepository(repositoryID); err != nil {
							log.Printf("stashkins.main failed to delete Maven repository %s: %+v\n", repositoryID, err)
						} else {
							log.Printf("Deleted Maven repositoryID %s\n", repositoryID)
							if err := mavenRepositoryClient.DeleteRepositoryFromGroup(repositoryID, *mavenRepositoryGroupID); err != nil {
								log.Printf("stashkins.main failed to delete Maven repository %s from repository group %s: %+v\n", repositoryID, *mavenRepositoryGroupID, err)
							} else {
								log.Printf("Removed Maven repositoryID %s from repository groupID %s\n", repositoryID, *mavenRepositoryGroupID)
							}
						}
					}
				}
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
			log.Printf("Number of missing jobs: %d\n", len(missingJobs))

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
					log.Fatalf("stashkins.main cannot read job template file %s: %v\n", *jobTemplateFile, err)
				}
				jobTemplate, err := template.New("jobconfig").Parse(string(data))
				if err != nil {
					log.Fatalf("stashkins.main cannot parse job template file %s: %v\n", *jobTemplateFile, err)
				}
				result := bytes.NewBufferString("")
				err = jobTemplate.Execute(result, jobDescr)
				if err != nil {
					log.Fatalf("stashkins.main cannot execute job template file %s: %v\n", *jobTemplateFile, err)
				}
				templateString := string(result.Bytes())

				// Create the job
				err = jenkins.CreateJob(*jenkinsBaseURL, jobDescr.JobName, templateString)
				if err != nil {
					log.Printf("stashkins.main failed to create job %+v, continuing...: error==%+v\n", jobDescr, err)
				} else {
					log.Printf("created job %+v\n", jobDescr)
				}

				if *doMavenRepoManagement {
					branchRepresentation := strings.Replace(branch, "/", "_", -1)
					repositoryID := fmt.Sprintf("%s.%s.%s", repo.Project.Key, repo.Slug, branchRepresentation)
					if present, err := mavenRepositoryClient.RepositoryExists(repositoryID); err == nil && !present {
						if err := mavenRepositoryClient.CreateRepository(repositoryID); err != nil {
							log.Printf("stashkins.main failed to create Maven repository %s: %+v\n", repositoryID, err)
						} else {
							log.Printf("Created Maven repositoryID %s\n", repositoryID)
						}
					} else {
						if err != nil {
							log.Printf("stashkins.main error creating Maven repositoryID %s: %v\n", repositoryID, err)
						} else {
							log.Printf("stashkins.main Maven repositoryID %s exists.  Skipping.\n", repositoryID)
						}
					}
					if err := mavenRepositoryClient.AddRepositoryToGroup(repositoryID, *mavenRepositoryGroupID); err != nil {
						log.Printf("stashkins.main failed to add Maven repository %s to repository group %s: %+v\n", repositoryID, *mavenRepositoryGroupID, err)
					} else {
						log.Printf("Maven repositoryID %s added to repository groupID %s\n", repositoryID, *mavenRepositoryGroupID)
					}
				}
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
