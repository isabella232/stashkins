package stashkins

import (
	"fmt"
	"strings"
	"time"

	"unicode"

	"github.com/ae6rt/retry"
	"github.com/xoom/maventools"
)

const postCreatorAgent = "Maven postCreator"
const postDeleterAgent = "Maven postDeleter"

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
		Log.Printf("%s:  skipping tasks for non-feature branch %s:\n", postDeleterAgent, branch)
		return nil
	}

	repositoryID := maventools.RepositoryID(maven.repositoryID(templateRecord.ProjectKey, templateRecord.Slug, branch))
	if _, err := maven.client.DeleteRepository(repositoryID); err != nil {
		Log.Printf("%s: failed to delete Maven repository %v: %+v\n", postDeleterAgent, repositoryID, err)
		return err
	} else {
		Log.Printf("%s: deleted Maven repository %v\n", postDeleterAgent, repositoryID)
	}
	return nil
}

func (maven MavenAspect) PostJobCreateTasks(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord JobTemplate) error {

	if !maven.branchOperations.isFeatureBranch(branch) {
		Log.Printf("%s: skipping tasks for non-feature branch %s\n", postCreatorAgent, branch)
		return nil
	}

	repositoryID := maventools.RepositoryID(maven.repositoryID(templateRecord.ProjectKey, templateRecord.Slug, branch))
	if present, err := maven.client.RepositoryExists(repositoryID); err == nil && !present {
		if _, err := maven.client.CreateSnapshotRepository(repositoryID); err != nil {
			Log.Printf("%s: failed to create Maven repository %v: %+v\n", postCreatorAgent, repositoryID, err)
			return err
		} else {
			Log.Printf("%s: created Maven repositoryID %v\n", postCreatorAgent, repositoryID)
			// falls through to add the repository
		}
	} else if err != nil {
		Log.Printf("%s: error checking if Maven repositoryID %v exists: %v\n", postCreatorAgent, repositoryID, err)
		return err
	} else {
		Log.Printf("%s: Maven repositoryID %v exists.  Skipping.\n", postCreatorAgent, repositoryID)
		// we historically allow this to fall through and re-add the repository
	}

	if err := maven.waitForRepositoryToSettle(repositoryID); err != nil {
		Log.Printf("%s: per-branch repository %s does not exist or error trying to determine as much.\n", postCreatorAgent, err)
		return err
	}

	repositoryGroupID := maventools.GroupID(maven.mavenRepositoryParams.FeatureBranchRepositoryGroupID)
	if rc, err := maven.client.AddRepositoryToGroup(repositoryID, repositoryGroupID); err != nil {
		Log.Printf("%s: failed to add Maven repository %s to repository group %v: %+v\n", postCreatorAgent, repositoryID, maven.mavenRepositoryParams.FeatureBranchRepositoryGroupID, err)
		return err
	} else {
		if rc == 200 {
			Log.Printf("%s: repositoryID %v added to repository groupID %s\n", postCreatorAgent, repositoryID, maven.mavenRepositoryParams.FeatureBranchRepositoryGroupID)
		}
	}
	return nil
}

func (maven MavenAspect) waitForRepositoryToSettle(repositoryID maventools.RepositoryID) error {
	retry := retry.New(16*time.Second, 5, func(attempts uint) {
		if attempts == 0 {
			return
		}
		if attempts > 2 {
			Log.Printf("%s: wait for repository-exists with-backoff try %d\n", postCreatorAgent, attempts+1)
		}
		time.Sleep((1 << attempts) * time.Second)
	})

	// Sonatype says Nexus will perform asynchronous tasks on creating the repository after Nexus returns 201 Created above.  As a result, the repository
	// may not actually be eligible for addition to the group when the call to create returns.  So poll Nexus for a short time, waiting for the repository
	// to be fully formed, which Sonatype says is indicated by an HTTP 200 OK in response to an HTTP GET on the repository ID.
	work := func() error {
		exists, err := maven.client.RepositoryExists(repositoryID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Repository does not exist")
		}
		return nil
	}
	return retry.Try(work)
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
