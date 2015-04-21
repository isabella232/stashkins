package stashkins

import (
	"testing"

	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
)

func TestJobNotObsolete(t *testing.T) {
	s := StatelessOperations{}
	jobSummary := jenkins.JobSummary{Branch: "origin/feature/1"}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobNotObsoleteBranchUnmanaged(t *testing.T) {
	s := StatelessOperations{}
	jobSummary := jenkins.JobSummary{Branch: "origin/master"}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobObsolete(t *testing.T) {
	s := StatelessOperations{}
	jobSummary := jenkins.JobSummary{Branch: "origin/feature/1"}
	stashBranches := map[string]stash.Branch{"feature/2": stash.Branch{}}
	if !s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}
