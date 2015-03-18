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

	templates, err := getTemplates("foo")
	if err != nil {
		log.Fatalf("stashkins.main cannot fetch job templates:  %v\n", err)
	}
	for _, template := range templates {
		stashkins := stashkins.NewStashkins(stashParams, jenkinsParams, nexusParams)
		if err := stashkins.Run(template); err != nil {
			log.Printf("Error creating new jobs with template %#v\n", err)
			continue
		}
	}
}

func getTemplates(templateRepo string) ([]stashkins.Template, error) {
	repos := make([]stashkins.Template, 0)
	repos = append(repos, stashkins.Template{ProjectKey: "PLAT", Slug: "trunk", JobType: jenkins.Maven})
	repos = append(repos, stashkins.Template{ProjectKey: "PLAT", Slug: "xoom", JobType: jenkins.Maven})
	return repos, nil
}

func validateCommandLineArguments() {

	if *userName == "" {
		log.Fatalf("ldapUser is required")
	}

	if *password == "" {
		log.Fatalf("ldapPassword is required")
	}

	if *jobTemplateRepositoryURL == "" {
		log.Fatalf("template-repository-url is required")
	}

	if *mavenRepositoryGroupID == "" {
		log.Fatalf("maven-repo-repository-groupID is required")
	}

	if *mavenBaseURL == "" || *mavenUsername == "" || *mavenPassword == "" || *mavenRepositoryGroupID == "" {
		log.Fatalf("maven-repo-base-url, maven-repo-username, maven-repo-password, and maven-repo-repository-groupID are required\n")
	}
}
