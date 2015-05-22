package main

import (
	"flag"

	"log"
	"os"

	"github.com/xoom/jenkins"

	"github.com/xoom/stashkins/stashkins"
)

var (
	stashBaseURL             = flag.String("stash-rest-base-url", "http://stash.example.com:8080", "Stash REST Base URL")
	jenkinsBaseURL           = flag.String("jenkins-base-url", "http://jenkins.example.com:8080", "Jenkins Base URL")
	jobTemplateRepositoryURL = flag.String("job-template-repository-url", "", "The Stash repository where job templates are stored..")
	jobTemplateBranch        = flag.String("job-template-repository-branch", "master", "Templates are held a Stash repository.  This is the branch from which to fetch the job template.")
	userName                 = flag.String("username", "", "User capable of doing automation tasks on Stash and Jenkins")
	password                 = flag.String("password", "", "Password for automation user")
	mavenBaseURL             = flag.String("maven-repo-base-url", "http://localhost:8081/nexus", "Maven repository management Base URL")
	mavenUsername            = flag.String("maven-repo-username", "", "User capable of doing automation of Maven repository management")
	mavenPassword            = flag.String("maven-repo-password", "", "Password for Maven repository management user")
	mavenRepositoryGroupID   = flag.String("maven-repo-repository-groupID", "", "Repository groupID in which to group new per-branch repositories")
	managedBranchPrefixes    = flag.String("managed-branch-prefixes", "feature/", "Branch prefixes to manage.")
	versionFlag              = flag.Bool("version", false, "Print build info from which stashkins was built")

	Log *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	stashParams   stashkins.WebClientParams
	jenkinsParams stashkins.WebClientParams
	nexusParams   stashkins.MavenRepositoryParams

	buildInfo string
)

func init() {
	flag.Parse()
	stashParams = stashkins.WebClientParams{URL: *stashBaseURL, UserName: *userName, Password: *password}
	jenkinsParams = stashkins.WebClientParams{URL: *jenkinsBaseURL, UserName: *userName, Password: *password}
	nexusParams = stashkins.MavenRepositoryParams{
		WebClientParams: stashkins.WebClientParams{
			URL:      *mavenBaseURL,
			UserName: *mavenUsername,
			Password: *mavenPassword,
		},
		FeatureBranchRepositoryGroupID: *mavenRepositoryGroupID,
	}
}

func main() {
	log.Printf("%s\n", buildInfo)
	if *versionFlag {
		os.Exit(0)
	}

	validateCommandLineArguments()

	homeDirectory := os.Getenv("HOME")
	if homeDirectory == "" {
		Log.Fatalf("main: HOME environment variable must be set to locate local template repository clone.\n")
	}
	Log.Printf("HOME: %+v\n", homeDirectory)

	branchOperations := stashkins.NewBranchOperations(*managedBranchPrefixes)

	skins := stashkins.NewStashkins(stashParams, jenkinsParams, nexusParams, branchOperations)

	jobSummaries, err := skins.GetJobSummaries()
	if err != nil {
		Log.Fatalf("main: Cannot get Jenkins job summaries: %#v\n", err)
	}

	jobTemplates, err := stashkins.GetTemplates(*jobTemplateRepositoryURL, *jobTemplateBranch, homeDirectory+"/stashkins-work")
	if err != nil {
		Log.Fatalf("main: cannot fetch job templates:  %v\n", err)
	}

	for _, jobTemplate := range jobTemplates {
		var jobAspect stashkins.Aspect

		switch jobTemplate.JobType {
		case jenkins.Maven:
			jobAspect = stashkins.NewMavenAspect(nexusParams, skins.NexusClient, branchOperations)
		case jenkins.Freestyle:
			jobAspect = stashkins.NewFreestyleAspect()
		}

		Log.Printf("Reconciling jobs for %s/%s\n", jobTemplate.ProjectKey, jobTemplate.Slug)
		if err := skins.ReconcileJobs(jobSummaries, jobTemplate, jobAspect); err != nil {
			Log.Printf("main: error reconciling jobs for %s/%s: %#v\n", jobTemplate.ProjectKey, jobTemplate.Slug, err)
		}
	}
}

func validateCommandLineArguments() {

	if *userName == "" || *password == "" {
		Log.Fatalf("username and password are required")
	}

	if *jobTemplateRepositoryURL == "" {
		Log.Fatalf("template-repository-url is required")
	}

	if *mavenRepositoryGroupID == "" {
		Log.Fatalf("maven-repo-repository-groupID is required")
	}

	if *mavenUsername == "" || *mavenPassword == "" || *mavenRepositoryGroupID == "" {
		Log.Fatalf("maven-repo-username, maven-repo-password, and maven-repo-repository-groupID are required\n")
	}
}
