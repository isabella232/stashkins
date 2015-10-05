package stashkins

import (
	"fmt"
	"strings"
	"time"

	"unicode"

	"github.com/xoom/maventools"
)

type MavenAspect struct {
	mavenRepositoryParams MavenRepositoryParams
	client                maventools.NexusClient
	branchOperations      BranchOperations
	Aspect
}

func NewMavenAspect(params MavenRepositoryParams, client maventools.NexusClient, branchOperations BranchOperations) Aspect {
	return MavenAspect{mavenRepositoryParams: params, client: client, branchOperations: branchOperations}
}

func (maven MavenAspect) MakeModel(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord JobTemplate) interface{} {
	return MavenJob{
		JobName:                    newJobName,
		Description:                newJobDescription,
		BranchName:                 branch,
		RepositoryURL:              gitRepositoryURL,
		MavenSnapshotRepositoryURL: maven.repositoryURL(templateRecord.ProjectKey, templateRecord.Slug, branch),
		MavenRepositoryID:          maven.repositoryID(templateRecord.ProjectKey, templateRecord.Slug, branch),
	}
}

func (maven MavenAspect) PostJobDeleteTasks(jobName, gitRepositoryURL, branch string, templateRecord JobTemplate) error {
	if !maven.branchOperations.isFeatureBranch(branch) {
		Log.Printf("Maven postDeleter skipping tasks for non-feature branch %s:\n", branch)
		return nil
	}

	repositoryID := maventools.RepositoryID(maven.repositoryID(templateRecord.ProjectKey, templateRecord.Slug, branch))
	if _, err := maven.client.DeleteRepository(repositoryID); err != nil {
		Log.Printf("Maven postDeleter failed to delete Maven repository %v: %+v\n", repositoryID, err)
		return err
	} else {
		Log.Printf("Maven postDeleter deleted Maven repository %v\n", repositoryID)
	}
	return nil
}

func (maven MavenAspect) PostJobCreateTasks(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord JobTemplate) error {
	if !maven.branchOperations.isFeatureBranch(branch) {
		Log.Printf("Maven postCreator skipping tasks for non-feature branch %s\n", branch)
		return nil
	}

	repositoryID := maventools.RepositoryID(maven.repositoryID(templateRecord.ProjectKey, templateRecord.Slug, branch))
	if present, err := maven.client.RepositoryExists(repositoryID); err == nil && !present {
		if rc, err := maven.client.CreateSnapshotRepository(repositoryID); err != nil {
			Log.Printf("Maven postcreator failed to create Maven repository %v: %+v\n", repositoryID, err)
			return err
		} else {
			if rc == 201 {
				Log.Printf("Maven postCreator created Maven repositoryID %v\n", repositoryID)
				const sleepy time.Duration = 3
				time.Sleep(sleepy * time.Second)
				Log.Printf("Slept for %d seconds before adding repository to per-branch group\n", sleepy)
			}
		}
	} else {
		if err != nil {
			Log.Printf("Maven postCreator: error checking if Maven repositoryID %v exists: %v\n", repositoryID, err)
			return err
		} else {
			Log.Printf("Maven postCreator: Maven repositoryID %v exists.  Skipping.\n", repositoryID)
		}
	}

	repositoryGroupID := maventools.GroupID(maven.mavenRepositoryParams.FeatureBranchRepositoryGroupID)
	if rc, err := maven.client.AddRepositoryToGroup(repositoryID, repositoryGroupID); err != nil {
		Log.Printf("Maven postCreator: failed to add Maven repository %s to repository group %v: %+v\n", repositoryID, maven.mavenRepositoryParams.FeatureBranchRepositoryGroupID, err)
		return err
	} else {
		if rc == 200 {
			Log.Printf("Maven repositoryID %v added to repository groupID %s\n", repositoryID, maven.mavenRepositoryParams.FeatureBranchRepositoryGroupID)
		}
	}
	return nil
}

func (maven MavenAspect) repositoryURL(gitProjectKey, gitRepositorySlug, gitBranch string) string {
	var mavenSnapshotRepositoryURL string
	if gitBranch == "develop" {
		mavenSnapshotRepositoryURL = fmt.Sprintf("%s/content/repositories/snapshots", maven.mavenRepositoryParams.URL)
	} else {
		// For feature/ branches, use per-branch repositories
		mavenSnapshotRepositoryID := maven.repositoryID(gitProjectKey, gitRepositorySlug, gitBranch)
		mavenSnapshotRepositoryURL = fmt.Sprintf("%s/content/repositories/%s", maven.mavenRepositoryParams.URL, mavenSnapshotRepositoryID)
	}
	return mavenSnapshotRepositoryURL
}

func (maven MavenAspect) repositoryID(gitRepoProjectKey, gitRepoSlug, gitBranch string) string {
	branch := maven.branchOperations.stripLeadingOrigin(gitBranch)
	if branch == "develop" {
		return "snapshots"
	}
	return maven.scrubRepositoryID(fmt.Sprintf("%s.%s.%s", gitRepoProjectKey, gitRepoSlug, branch))
}

// Only letters, digits, underscores(_), hyphens(-), and dots(.) are allowed in a repository ID, per the Nexus web UI.  Replace
// disallowed characters with _.
func (maven MavenAspect) scrubRepositoryID(in string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '_'
	}, in)
}
