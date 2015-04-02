package stashkins

import (
	"fmt"
	"strings"

	"github.com/xoom/maventools"
)

type MavenAspect struct {
	MavenRepositoryParams MavenRepositoryParams
	Client                maventools.Client
	StatelessOperations
	Aspect
}

func NewMavenAspect(params MavenRepositoryParams, client maventools.Client) Aspect {
	return MavenAspect{MavenRepositoryParams: params, Client: client}
}

func (maven MavenAspect) MakeModel(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord Template) interface{} {
	mavenSnapshotRepositoryURL := maven.buildMavenRepositoryURL(templateRecord.ProjectKey, templateRecord.Slug, branch)

	return MavenJob{
		JobName:                    newJobName,
		Description:                newJobDescription,
		BranchName:                 branch,
		RepositoryURL:              gitRepositoryURL,
		MavenSnapshotRepositoryURL: mavenSnapshotRepositoryURL,
	}
}

func (maven MavenAspect) PostJobDeleteTasks(jobName, gitRepositoryURL, branchName string, templateRecord Template) error {
	if !maven.isFeatureBranch(branchName) {
		Log.Printf("maven postdelete skipping tasks for non-feature branch %s:\n", branchName)
		return nil
	}

	var branchRepresentation string
	if strings.HasPrefix(branchName, "origin/") {
		branchRepresentation = branchName[len("origin/"):]
	}
	branchRepresentation = strings.Replace(branchRepresentation, "/", "_", -1)
	repositoryID := maventools.RepositoryID(fmt.Sprintf("%s.%s.%s", templateRecord.ProjectKey, templateRecord.Slug, branchRepresentation))
	if _, err := maven.Client.DeleteRepository(repositoryID); err != nil {
		Log.Printf("Maven postDeleter failed to delete Maven repository %s: %+v\n", repositoryID, err)
		return err
	} else {
		Log.Printf("Deleted Maven repository %v\n", repositoryID)
	}
	return nil
}

func (maven MavenAspect) PostJobCreateTasks(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord Template) error {
	if !maven.isFeatureBranch(branch) {
		Log.Printf("maven postcreator skipping tasks for non-feature branch %s:\n", branch)
		return nil
	}

	branchRepresentation := strings.Replace(branch, "/", "_", -1)
	repositoryID := maventools.RepositoryID(fmt.Sprintf("%s.%s.%s", templateRecord.ProjectKey, templateRecord.Slug, branchRepresentation))
	if present, err := maven.Client.RepositoryExists(repositoryID); err == nil && !present {
		if rc, err := maven.Client.CreateSnapshotRepository(repositoryID); err != nil {
			Log.Printf("Maven postcreator failed to create Maven repository %s: %+v\n", repositoryID, err)
			return err
		} else {
			if rc == 201 {
				Log.Printf("Created Maven repositoryID %s\n", repositoryID)
			}
		}
	} else {
		if err != nil {
			Log.Printf("Maven postCreator: error checking if Maven repositoryID %s exists: %v\n", repositoryID, err)
			return err
		} else {
			Log.Printf("Maven postCreator: Maven repositoryID %s exists.  Skipping.\n", repositoryID)
		}
	}

	repositoryGroupID := maventools.GroupID(maven.MavenRepositoryParams.PerBranchRepositoryID)
	if rc, err := maven.Client.AddRepositoryToGroup(repositoryID, repositoryGroupID); err != nil {
		Log.Printf("Maven postCreator: failed to add Maven repository %s to repository group %s: %+v\n", repositoryID, maven.MavenRepositoryParams.PerBranchRepositoryID, err)
		return err
	} else {
		if rc == 200 {
			Log.Printf("Maven repositoryID %s added to repository groupID %s\n", repositoryID, maven.MavenRepositoryParams.PerBranchRepositoryID)
		}
	}
	return nil
}

func (maven MavenAspect) buildMavenRepositoryURL(gitProjectKey, gitRepositorySlug, gitBranch string) string {
	var mavenSnapshotRepositoryURL string
	if gitBranch == "develop" {
		mavenSnapshotRepositoryURL = fmt.Sprintf("%s/content/repositories/snapshots", maven.MavenRepositoryParams.URL)
	} else {
		// For feature/ branches, use per-branch repositories
		mavenSnapshotRepositoryID := maven.mavenRepositoryID(gitProjectKey, gitRepositorySlug, gitBranch)
		mavenSnapshotRepositoryURL = fmt.Sprintf("%s/content/repositories/%s", maven.MavenRepositoryParams.URL, mavenSnapshotRepositoryID)
	}
	return mavenSnapshotRepositoryURL
}

func (maven MavenAspect) mavenRepositoryID(gitRepoProjectKey, gitRepoSlug, gitBranch string) string {
	return fmt.Sprintf("%s.%s.%s", maven.mavenRepoIDPartCleaner(gitRepoProjectKey), maven.mavenRepoIDPartCleaner(gitRepoSlug), maven.mavenRepoIDPartCleaner(gitBranch))
}

func (maven MavenAspect) mavenRepoIDPartCleaner(b string) string {
	thing := b
	thing = strings.Replace(thing, "/", "_", -1)
	thing = strings.Replace(thing, "&", "_", -1)
	thing = strings.Replace(thing, "?", "_", -1)
	return thing
}
