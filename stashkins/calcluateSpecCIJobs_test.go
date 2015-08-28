package stashkins

import (
	"github.com/xoom/stash"
	"testing"
)

// Calculate the job names which must exist per the underlying managed branch names
func TestCalculateSpecCIJobs(t *testing.T) {
	skins := NewStashkins(WebClientParams{}, WebClientParams{}, MavenRepositoryParams{}, NewBranchOperations("feature/,hotfix/"))

	// setup reference data
	displayIDs := []string{"master", "other", "develop", "feature/issue-88", "hotfix/issue-99", "nope/issue-100"}
	branches := make(map[string]stash.Branch)
	for _, v := range displayIDs {
		branches[v] = stash.Branch{DisplayID: v}
	}

	// Calculate spec CI jobs
	specJobs := skins.calculateSpecCIJobs("proj", "somelib", branches)

	if len(specJobs) != 3 {
		t.Fatalf("Want 3 but got %d\n", len(specJobs))
	}

	// Verify that each expected job name is in the spec list of job names
	for _, expectedJobName := range []string{
		"proj-somelib-continuous-develop",
		"proj-somelib-continuous-feature-issue-88",
		"proj-somelib-continuous-hotfix-issue-99",
	} {
		var foundIt bool = false
		for _, jobName := range specJobs {
			if jobName == expectedJobName {
				foundIt = true
				break
			}
		}
		if !foundIt {
			t.Fatalf("Spec job name %s not found\n", expectedJobName)
		}
	}
}
