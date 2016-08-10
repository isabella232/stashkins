package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os/signal"
	"runtime"
	"syscall"

	"log"
	"os"

	"github.com/xoom/jenkins"

	"strings"

	"github.com/xoom/stashkins/stashkins"
)

var (
	stashBaseURL             = flag.String("stash-rest-base-url", "http://stash.example.com:8080", "Stash REST Base URL")
	jenkinsBaseURL           = flag.String("jenkins-base-url", "http://jenkins.example.com:8080", "Jenkins Base URL")
	jenkinsJobsDirectory     = flag.String("jenkins-jobs-directory", "", "Filesystem location of Jenkins jobs directory.  Used when acquiring job summaries from the Jenkins master filesystem.")
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

	Log *log.Logger = log.New(os.Stdout, "stashkins ", log.Ldate|log.Ltime|log.Lshortfile)

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
	Log.Printf("%s\n", buildInfo)
	if *versionFlag {
		os.Exit(0)
	}

	// Setup a lock file so consecutive runs do not overlap
	if runtime.GOOS == "linux" {
		// https://github.com/golang/go/issues/8456
		lock, err := os.OpenFile("/var/lock/stashkins.lock", os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			Log.Println(err)
			return
		}
		defer lock.Close()

		err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			Log.Printf("Error acquiring lock on %s: %v\n", lock.Name(), err)
			return
		}

		go func(f *os.File) {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			f.Close()
			os.Remove(f.Name())
			os.Exit(1)
		}(lock)

		defer func(f *os.File) {
			if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
				Log.Printf("Error releasing lock on %s: %v\n", f.Name(), err)
				return
			}
			if err := os.Remove(f.Name()); err != nil {
				Log.Printf("Error removing lock file %s: %v\n", f.Name(), err)
			}
		}(lock)
	}

	Log.Println("Stashkins __begin")

	if err := validateCommandLineArguments(); err != nil {
		Log.Println(err)
		return
	}

	branchOperations := stashkins.NewBranchOperations(*managedBranchPrefixes)

	skins := stashkins.NewStashkins(stashParams, jenkinsParams, nexusParams, branchOperations)

	var jobSummaries []jenkins.JobSummary

	var err error
	if *jenkinsJobsDirectory == "" {
		jobSummaries, err = skins.JobSummariesOverHTTP()
		if err != nil {
			Log.Printf("main: Cannot get Jenkins job summaries over HTTP: %#v\n", err)
			return
		}
	} else {
		jobSummaries, err = skins.JobSummariesFromFilesystem(*jenkinsJobsDirectory)
		if err != nil {
			Log.Printf("main: Cannot get Jenkins job summaries from filesystem: %#v\n", err)
			return
		}
	}
	Log.Printf("Found %d Jenkins job summaries\n", len(jobSummaries))

	templateCloneDirectory, err := ioutil.TempDir("", "stashkins-templates-")
	if err != nil {
		Log.Fatalln(err)
	}
	defer func() {
		os.RemoveAll(templateCloneDirectory)
	}()

	jobTemplates, err := stashkins.Templates(*jobTemplateRepositoryURL, *jobTemplateBranch, templateCloneDirectory)
	if err != nil {
		Log.Printf("main: cannot fetch job templates:  %v\n", err)
		return
	}
	Log.Printf("Found %d Jenkins job templates\n", len(jobTemplates))

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
			Log.Printf("main: warning: while reconciling jobs for %s/%s: %v\n", jobTemplate.ProjectKey, jobTemplate.Slug, err)
		}
	}
	Log.Println("Stashkins has finished (__finish).")
}

func validateCommandLineArguments() error {
	if *userName == "" || *password == "" {
		return errors.New("username and password are required")
	}

	if *jobTemplateRepositoryURL == "" {
		return errors.New("template-repository-url is required")
	}

	if *mavenRepositoryGroupID == "" {
		return errors.New("maven-repo-repository-groupID is required")
	}

	if *mavenUsername == "" || *mavenPassword == "" || *mavenRepositoryGroupID == "" {
		return errors.New("maven-repo-username, maven-repo-password, and maven-repo-repository-groupID are required")
	}

	if *jenkinsJobsDirectory != "" && !strings.HasPrefix(*jenkinsJobsDirectory, "/") {
		return fmt.Errorf("jenkins-jobs-directory must be specified with an absolute path: %s\n", *jenkinsJobsDirectory)
	}
	return nil
}
