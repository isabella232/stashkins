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
	"github.com/xoom/maventools"
	"github.com/xoom/maventools/nexus"
	"github.com/xoom/stash"
)

// JobTemplate is used to populate a template XML Jenkins job config file with appropriate values for prospective new jobs
type JobTemplate struct {
	JobName                             string // code in ssh://git@example.com:9999/teamp/code.git
	Description                         string // mashup of repository URL and branch name
	BranchName                          string // feature/PROJ-999, as in feature/PROJ-999
	RepositoryURL                       string // ssh://git@example.com:9999/teamp/code.git
	NexusRepositoryType                 string // if branch == master then releases else snapshots
	PerBranchMavenSnapshotRepositoryID  string // the Maven repository ID to which to publish this job's artifacts
	PerBranchMavenSnapshotRepositoryURL string // the Maven repository URL to which to publish this job's artifacts
}

var (
	stashBaseURL   = flag.String("stash-rest-base-url", "http://stash.example.com:8080", "Stash REST Base URL")
	jenkinsBaseURL = flag.String("jenkins-url", "http://jenkins.example.com:8080", "Jenkins Base URL")

	jobTemplateFile = flag.String("job-template-file", "job-template.xml", "Jenkins job template file.")

	jobRepositoryProjectKey = flag.String("repository-project-key", "", "The Stash Project Key for the job-repository of interest.  For example, PLAYG.")
	jobRepositorySlug       = flag.String("repository-slug", "", "The Stash repository 'slug' for the job-repository of interest.  For example, 'trunk'.")

	stashUserName = flag.String("stash-username", "", "Username for Stash authentication")
	stashPassword = flag.String("stash-password", "", "Password for Stash authentication")

	mavenBaseURL           = flag.String("maven-repo-base-url", "http://localhost:8081/nexus", "Maven repository management Base URL")
	mavenUsername          = flag.String("maven-repo-username", "", "Username for Maven repository management")
	mavenPassword          = flag.String("maven-repo-password", "", "Password for Maven repository management")
	mavenRepositoryGroupID = flag.String("maven-repo-repository-groupID", "", "Repository groupID in which to group new per-branch repositories")
	doNexus                = flag.Bool("do-nexus", false, "Do Maven repository management against Sonatype Nexus.  Precludes do-artifactory.")
	doArtifactory          = flag.Bool("do-artifactory", false, "Do Maven repository management against JFrog Artifactory.  Precludes do-nexus.")

	versionFlag = flag.Bool("version", false, "Print build info from which stashkins was built")

	mavenRepositoryClient maventools.Client

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

	validateCommandLineArguments()

	if *doNexus {
		mavenRepositoryClient = nexus.NewClient(*mavenBaseURL, *mavenUsername, *mavenPassword)
	}

	doMavenRepoManagement := *doNexus || *doArtifactory

	// Fetch repository metadata
	repo, err := stash.GetRepository(*stashBaseURL, *stashUserName, *stashPassword, *jobRepositoryProjectKey, *jobRepositorySlug)
	if err != nil {
		log.Fatalf("stashkins.main GetRepository error %v\n", err)
	}
	jobRepositoryURL := repo.SshUrl()
	if jobRepositoryURL == "" {
		log.Fatalf("No SSH based URL for this repository")
	}
	log.Printf("Analyzing repository %s...\n", jobRepositoryURL)

	// Fetch all branches for this repository
	stashBranches, err := stash.GetBranches(*stashBaseURL, *stashUserName, *stashPassword, repo.Project.Key, repo.Slug)
	if err != nil {
		log.Fatalf("stashkins.main error getting branches from Stash for repository %s: %v\n", jobRepositoryURL, err)
	}

	// Fetch all Jenkins jobs
	allJobs, err := jenkins.GetJobs(*jenkinsBaseURL)
	if err != nil {
		log.Fatalf("stashkins.main get jobs error: %v\n", err)
	}

	// Filter on jobs that build against a branch of the specified git repository
	targetJobs := make([]jenkins.JobConfig, 0)
	for _, job := range allJobs {
		jobConfig, err := jenkins.GetJobConfig(*jenkinsBaseURL, job.Name)
		if err != nil {
			// This probably means the job config is not a maven job.
			log.Printf("stashkins.main Jenkins GetJobConfig error (not a Maven job?) for job %s: %v.  Skipping.\n", job.Name, err)
		}
		if len(jobConfig.SCM.UserRemoteConfigs.UserRemoteConfig) != 1 {
			log.Printf("The job %s does not have exactly one UserRemoteConfig.  Skipping.\n", job.Name)
			continue
		}

		remoteCfg := jobConfig.SCM.UserRemoteConfigs.UserRemoteConfig[0]
		if strings.HasPrefix(remoteCfg.URL, "http") {
			log.Printf("Job %s references an HTTP Git URL.  Only SSH Git URLs are supported.\n", job.Name)
			continue
		}
		if remoteCfg.URL == jobRepositoryURL {
			targetJobs = append(targetJobs, jobConfig)
		}
	}

	// Find branches Jenkins is building that no longer exist in Stash.  The branch the job is building must contain "feature/".
	obsoleteJobs := make([]jenkins.JobConfig, 0)
	for _, jobConfig := range targetJobs {
		if len(jobConfig.SCM.Branches.Branch) != 1 {
			log.Printf("The job %s builds more than one branch, which is unsupported.  Skipping job.\n", jobConfig.JobName)
			continue
		}
		builtBranch := jobConfig.SCM.Branches.Branch[0]
		if !branchIsManaged(builtBranch.Name) {
			continue
		}
		deleteJobConfig := true
		for stashBranch, _ := range stashBranches {
			if strings.HasSuffix(builtBranch.Name, stashBranch) {
				deleteJobConfig = false
			}
		}
		if deleteJobConfig {
			obsoleteJobs = append(obsoleteJobs, jobConfig)
		}
	}

	// Remove obsolete jobs
	log.Printf("Number of obsolete jobs: %d\n", len(obsoleteJobs))
	for _, job := range obsoleteJobs {
		if err := jenkins.DeleteJob(*jenkinsBaseURL, job.JobName); err != nil {
			log.Printf("stashkins.main error deleting obsolete job %s, continuing:  %+v\n", job.JobName, err)
		} else {
			log.Printf("Deleting obsolete job %+v\n", job.JobName)
		}

		// Maven repo management
		if doMavenRepoManagement {
			for _, branch := range job.SCM.Branches.Branch {
				var branchRepresentation string
				if strings.HasPrefix(branch.Name, "origin/") {
					branchRepresentation = branch.Name[len("origin/"):]
				}
				branchRepresentation = strings.Replace(branchRepresentation, "/", "_", -1)
				repositoryID := maventools.RepositoryID(fmt.Sprintf("%s.%s.%s", repo.Project.Key, repo.Slug, branchRepresentation))
				if _, err := mavenRepositoryClient.DeleteRepository(repositoryID); err != nil {
					log.Printf("stashkins.main failed to delete Maven repository %s: %+v\n", repositoryID, err)
				} else {
					log.Printf("Deleted Maven repository %v\n", repositoryID)
				}
			}
		}
	}

	// Find missing jobs.  This is characterized as a branch in Stash that is not built by any job.  The outstanding Stash branch must contain "feature/".
	missingJobs := make([]string, 0)
	for branch, _ := range stashBranches {
		if !branchIsManaged(branch) {
			continue
		}
		missingJob := true
		for _, jobConfig := range targetJobs {
			if len(jobConfig.SCM.Branches.Branch) != 1 {
				log.Printf("The job %s builds more than one branch, which is unsupported.  Skipping job.\n", jobConfig.JobName)
				continue
			}
			builtBranch := jobConfig.SCM.Branches.Branch[0]
			if strings.HasSuffix(builtBranch.Name, branch) {
				missingJob = false
			}
		}
		if missingJob {
			missingJobs = append(missingJobs, branch)
		}
	}

	// Create missing Jenkins jobs
	log.Printf("Number of missing jobs: %d\n", len(missingJobs))
	for _, branch := range missingJobs {
		branchType, branchSuffix := suffixer(branch)

		// Forms the deploy-target Maven repository ID, from which a custom settings.xml can be crafted.
		mavenSnapshotRepositoryID := mavenRepositoryID(repo.Project.Key, repo.Slug, branch)
		mavenSnapshotRepositoryURL := fmt.Sprintf("%s/content/repositories/%s", *mavenBaseURL, mavenSnapshotRepositoryID)

		jobDescr := JobTemplate{
			JobName:                             repo.Slug + "-continuous-" + branchType + branchSuffix,
			Description:                         "This is a continuous build for " + repo.Slug + ", branch " + branch,
			BranchName:                          branch,
			RepositoryURL:                       jobRepositoryURL,
			NexusRepositoryType:                 "snapshots",
			PerBranchMavenSnapshotRepositoryID:  mavenSnapshotRepositoryID,
			PerBranchMavenSnapshotRepositoryURL: mavenSnapshotRepositoryURL,
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
			log.Printf("created job %s\n", jobDescr.JobName)
		}

		// Maven repo management
		if doMavenRepoManagement {
			branchRepresentation := strings.Replace(branch, "/", "_", -1)
			repositoryID := maventools.RepositoryID(fmt.Sprintf("%s.%s.%s", repo.Project.Key, repo.Slug, branchRepresentation))
			if present, err := mavenRepositoryClient.RepositoryExists(repositoryID); err == nil && !present {
				if rc, err := mavenRepositoryClient.CreateSnapshotRepository(repositoryID); err != nil {
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
			repositoryGroupID := maventools.GroupID(*mavenRepositoryGroupID)
			if rc, err := mavenRepositoryClient.AddRepositoryToGroup(repositoryID, repositoryGroupID); err != nil {
				log.Printf("stashkins.main failed to add Maven repository %s to repository group %s: %+v\n", repositoryID, *mavenRepositoryGroupID, err)
			} else {
				if rc == 200 {
					log.Printf("Maven repositoryID %s added to repository groupID %s\n", repositoryID, *mavenRepositoryGroupID)
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

// Form the maven repository ID from project parts.  Each part must be cleaned and made URL-safe because the result will form part of an HTTP URL.
func mavenRepositoryID(gitRepoProjectKey, gitRepoSlug, gitBranch string) string {
	return fmt.Sprintf("%s.%s.%s", mavenRepoIDPartCleaner(gitRepoProjectKey), mavenRepoIDPartCleaner(gitRepoSlug), mavenRepoIDPartCleaner(gitBranch))
}

func mavenRepoIDPartCleaner(b string) string {
	thing := b
	thing = strings.Replace(thing, "/", "_", -1)
	thing = strings.Replace(thing, "&", "_", -1)
	thing = strings.Replace(thing, "?", "_", -1)
	return thing
}

func branchIsManaged(stashBranch string) bool {
	return strings.Contains(stashBranch, "feature/")
}

func validateCommandLineArguments() {
	if *jobRepositoryProjectKey == "" {
		log.Fatalf("repository-project-key must be set\n")
	}

	if *jobRepositorySlug == "" {
		log.Fatalf("repository-slug must be set.\n")
	}

	if *doNexus && *doArtifactory {
		log.Fatalf("Only one of do-nexus or do-artifactory may be set.\n")
	}

	if *doArtifactory {
		log.Fatalf("Artifactory is not supported yet")
	}

	if (*doNexus || *doArtifactory) && (*mavenBaseURL == "" || *mavenUsername == "" || *mavenPassword == "" || *mavenRepositoryGroupID == "") {
		log.Fatalf("maven-repo-base-url, maven-repo-username, maven-repo-password, and maven-repo-repository-groupID are required\n")
	}
}
