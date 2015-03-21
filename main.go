package main

import (
	"flag"

	"log"
	"os"

	"github.com/xoom/jenkins"

	"github.com/xoom/stashkins/stashkins"
)

var (
	stashBaseURL   = flag.String("stash-rest-base-url", "http://stash.example.com:8080", "Stash REST Base URL")
	jenkinsBaseURL = flag.String("jenkins-base-url", "http://jenkins.example.com:8080", "Jenkins Base URL")

	jobTemplateRepositoryURL = flag.String("job-template-repository-url", "", "The Stash repository where job templates are stored..")
	jobTemplateBranch        = flag.String("job-template-repository-branch", "master", "Templates are held a Stash repository.  This is the branch from which to fetch the job template.")

	userName = flag.String("username", "", "User capable of doing automation tasks on Stash and Jenkins")
	password = flag.String("password", "", "Password for automation user")

	mavenBaseURL           = flag.String("maven-repo-base-url", "http://localhost:8081/nexus", "Maven repository management Base URL")
	mavenUsername          = flag.String("maven-repo-username", "", "User capable of doing automation of Maven repository management")
	mavenPassword          = flag.String("maven-repo-password", "", "Password for Maven repository management user")
	mavenRepositoryGroupID = flag.String("maven-repo-repository-groupID", "", "Repository groupID in which to group new per-branch repositories")

	versionFlag = flag.Bool("version", false, "Print build info from which stashkins was built")

	version   string
	commit    string
	buildTime string
	sdkInfo   string
)

var stashParams stashkins.WebClientParams
var nexusParams stashkins.MavenRepositoryParams
var jenkinsParams stashkins.WebClientParams

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
		PerBranchRepositoryID: *mavenRepositoryGroupID,
	}
}

func main() {
	log.Printf("Version: %s, CommitID: %s, build time: %s, SDK Info: %s\n", version, commit, buildTime, sdkInfo)
	if *versionFlag {
		os.Exit(0)
	}

	validateCommandLineArguments()

	templates, err := getTemplates(*jobTemplateRepositoryURL, *jobTemplateBranch)
	if err != nil {
		log.Fatalf("stashkins.main cannot fetch job templates:  %v\n", err)
	}

	skins := stashkins.NewStashkins(stashParams, jenkinsParams, nexusParams)

	jobSummaries, err := skins.GetJobSummaries()
	if err != nil {
		log.Fatalf("Cannot get Jenkins job summaries: %#v\n", err)
	}
	for _, template := range templates {
		var jobAspect stashkins.Aspect

		switch template.JobType {
		case jenkins.Maven:
			jobAspect = stashkins.MavenAspect{MavenRepositoryParams: nexusParams, Client: skins.NexusClient}
		case jenkins.Freestyle:
			panic("Freestyle jobs not supported yet")
		}

		if err := skins.ReconcileJobs(jobSummaries, template, jobAspect); err != nil {
			log.Printf("Error reconciling jobs with template %#v\n", err)
			continue
		}
	}
}

func getTemplates(templateRepoURL, templateBranch string) ([]stashkins.Template, error) {
	repos := make([]stashkins.Template, 0)
	repos = append(repos, stashkins.Template{ProjectKey: "PROJ", Slug: "p1", JobType: jenkins.Maven})
	repos = append(repos, stashkins.Template{ProjectKey: "PROJ", Slug: "p2", JobType: jenkins.Maven})
	return repos, nil
}

func validateCommandLineArguments() {

	if *userName == "" || *password == "" {
		log.Fatalf("username and password are required")
	}

	if *jobTemplateRepositoryURL == "" {
		log.Fatalf("template-repository-url is required")
	}

	if *mavenRepositoryGroupID == "" {
		log.Fatalf("maven-repo-repository-groupID is required")
	}

	if *mavenUsername == "" || *mavenPassword == "" || *mavenRepositoryGroupID == "" {
		log.Fatalf("maven-repo-username, maven-repo-password, and maven-repo-repository-groupID are required\n")
	}
}
