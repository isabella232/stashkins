package stashkins

import (
	"github.com/xoom/stash"
	"testing"
)

func TestCanonicalCIJobName(t *testing.T) {
	branch := stash.Branch{
		DisplayID: "feature/PROJ-123",
	}

	jobName := BranchOperations{}.canonicalCIJobName("proj", "slug", branch)
	if jobName != "proj-slug-continuous-feature-PROJ-123" {
		t.Fatalf("Want proj-slug-continuous-feature-PROJ-123 but got %s\n", jobName)
	}
}
