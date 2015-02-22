package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
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

	stashURL, err := url.Parse(*stashBaseURL)
	if err != nil {
		log.Fatalf("Error parsing Stash base URL: %v\n", err)
	}
	stashClient := stash.NewClient(*stashUserName, *stashPassword, stashURL)

	jenkinsURL, err := url.Parse(*jenkinsBaseURL)
	if err != nil {
		log.Fatalf("Error parsing Jenkins base URL: %v\n", err)
	}
	jenkinsClient := jenkins.NewClient(jenkinsURL)

	if *doNexus {
		mavenRepositoryClient = nexus.NewClient(*mavenBaseURL, *mavenUsername, *mavenPassword)
	}

	doMavenRepoManagement := *doNexus || *doArtifactory

	// Fetch repository metadata
	repo, err := stashClient.GetRepository(*jobRepositoryProjectKey, *jobRepositorySlug)
	if err != nil {
		log.Fatalf("stashkins.main GetRepository error %v\n", err)
	}
	jobRepositoryURL := repo.SshUrl()
	if jobRepositoryURL == "" {
		log.Fatalf("No SSH based URL for this repository")
	}
	log.Printf("Analyzing repository %s...\n", jobRepositoryURL)

	// Fetch all branches for this repository
	stashBranches, err := stashClient.GetBranches(repo.Project.Key, repo.Slug)
	if err != nil {
		log.Fatalf("stashkins.main error getting branches from Stash for repository %s: %v\n", jobRepositoryURL, err)
	}

	// Fetch all Jenkins jobs
	allJobs, err := jenkinsClient.GetJobs()
	if err != nil {
		log.Fatalf("stashkins.main get jobs error: %v\n", err)
	}

	// Filter on jobs that build against the specified git repository
	targetJobs := make([]jenkins.JobConfig, 0)
	for _, job := range allJobs {
		jobConfig, err := jenkinsClient.GetJobConfig(job.Name)
		if err != nil {
			if !isIgnoreable(err) {
				log.Printf("stashkins.main Jenkins GetJobConfig error for job %s: %v.  Skipping.\n", job.Name, err)
			}
			continue
		}
		if isTargetJob(jobConfig.JobName, jobConfig.SCM.UserRemoteConfigs, jobRepositoryURL) {
			targetJobs = append(targetJobs, jobConfig)
		}
	}

	// Find branches Jenkins is building that no longer exist in Stash.  The branch the job is building must contain "feature/".
	obsoleteJobs := make([]jenkins.JobConfig, 0)
	for _, jobConfig := range targetJobs {
		if shouldDeleteJob(jobConfig, stashBranches) {
			obsoleteJobs = append(obsoleteJobs, jobConfig)
		}
	}

	// Remove obsolete jobs
	log.Printf("Number of obsolete jobs: %d\n", len(obsoleteJobs))
	for _, job := range obsoleteJobs {
		if err := jenkinsClient.DeleteJob(job.JobName); err != nil {
			log.Printf("stashkins.main error deleting obsolete job %s, continuing:  %+v\n", job.JobName, err)
		} else {
			log.Printf("Deleted obsolete job %+v\n", job.JobName)
		}

		// Maven repo management
		if doMavenRepoManagement {
			branch := job.SCM.Branches.Branch[0]
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

	// Find outstanding branches, the basis of which are Stash branches whose names contains "feature/" that are not being built by any job.
	branchesNeedingBuilding := make([]string, 0)
	for branch, _ := range stashBranches {
		if shouldCreateJob(targetJobs, branch) {
			branchesNeedingBuilding = append(branchesNeedingBuilding, branch)
		}
	}

	// Create Jenkins jobs to build outstanding branches
	log.Printf("Number of missing jobs: %d\n", len(branchesNeedingBuilding))
	for _, branch := range branchesNeedingBuilding {
		// For a branch feature/12, branchType will be "feature" and branchSuffix will be "12"
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
		err = jenkinsClient.CreateJob(jobDescr.JobName, templateString)
		if err != nil {
			log.Printf("stashkins.main failed to create job %+v, continuing...: error==%+v\n", jobDescr, err)
		} else {
			log.Printf("Created job %s\n", jobDescr.JobName)
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

func isTargetJob(jobName string, remoteConfigs jenkins.UserRemoteConfigs, jobRepositoryURL string) bool {
	if len(remoteConfigs.UserRemoteConfig) != 1 {
		log.Printf("The job %s does not have exactly one UserRemoteConfig.  Skipping.\n", jobName)
		return false
	}
	remoteCfg := remoteConfigs.UserRemoteConfig[0]
	if strings.HasPrefix(remoteCfg.URL, "http") {
		log.Printf("Job %s references an HTTP Git URL.  Only SSH Git URLs are supported.\n", jobName)
		return false
	}
	return remoteCfg.URL == jobRepositoryURL
}

func shouldDeleteJob(jobConfig jenkins.JobConfig, stashBranches map[string]stash.Branch) bool {
	if len(jobConfig.SCM.Branches.Branch) != 1 {
		log.Printf("The job %s builds more than one branch, which is unsupported.  Skipping job.\n", jobConfig.JobName)
		return false
	}
	builtBranch := jobConfig.SCM.Branches.Branch[0]
	if !branchIsManaged(builtBranch.Name) {
		return false
	}
	deleteJobConfig := true
	for stashBranch, _ := range stashBranches {
		if strings.HasSuffix(builtBranch.Name, stashBranch) {
			deleteJobConfig = false
		}
	}
	return deleteJobConfig
}

func shouldCreateJob(targetJobs []jenkins.JobConfig, branch string) bool {
	if !branchIsManaged(branch) {
		return false
	}
	for _, jobConfig := range targetJobs {
		if len(jobConfig.SCM.Branches.Branch) != 1 {
			log.Printf("The job %s builds more than one branch, which is unsupported.  Skipping job.\n", jobConfig.JobName)
			return false
		}
		builtBranch := jobConfig.SCM.Branches.Branch[0]
		if strings.HasSuffix(builtBranch.Name, branch) {
			return false
		}
	}
	return true
}

// I hate resorting to this, but there are no better solutions to suppress benign but scary look unmarshaling errors
// on jobs that are otherwise perfectly valid but are not maven jobs.  If they are not maven jobs, stashkins does not
// care about them.
func isIgnoreable(err error) bool {
	return strings.HasPrefix(err.Error(), "expected element type <maven2-moduleset> but have")
}
