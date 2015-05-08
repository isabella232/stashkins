package stashkins

import (
	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
	"testing"
)

func TestBranchIsManaged(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}

	if s.isBranchManaged("master") {
		t.Fatalf("want master managed == false but got true\n")
	}
	if !s.isBranchManaged("develop") {
		t.Fatalf("want develop managed == true but got true\n")
	}

	for _, prefix := range s.ManagedPrefixes {
		if !s.isBranchManaged(prefix + "somebranch") {
			t.Fatalf("want " + prefix + "somebranch managed == true but got false\n")
		}

		if !s.isBranchManaged("origin/" + prefix + "somebranch") {
			t.Fatalf("want origin/" + prefix + "somebranch managed == true but got false\n")
		}
	}

}

func TestIsFeatureBranch(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}

	if s.isFeatureBranch("master") {
		t.Fatalf("Want false\n")
	}

	if s.isFeatureBranch("develop") {
		t.Fatalf("Want false\n")
	}

	if !s.isFeatureBranch("origin/feature/1") {
		t.Fatalf("Want true\n")
	}

	if !s.isFeatureBranch("feature/1") {
		t.Fatalf("Want true\n")
	}

	if s.isFeatureBranch("origin/feature/*") {
		t.Fatalf("want false\n")
	}
}

func TestCreateLackingJob(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}
	jobSummaries := []jenkins.JobSummary{jenkins.JobSummary{Branch: "origin/feature/1"}}
	if !s.shouldCreateJob(jobSummaries, "feature/2") {
		t.Fatalf("Want true\n")
	}
}

func TestDoNotCreateJobAlreadyBeingBuilt(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}
	jobSummaries := []jenkins.JobSummary{jenkins.JobSummary{Branch: "origin/feature/1"}}
	if s.shouldCreateJob(jobSummaries, "feature/1") {
		t.Fatalf("Want false\n")
	}
}

func TestDoNotCreateJobForUnmanagedBranch(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}
	jobSummaries := make([]jenkins.JobSummary, 0)
	if s.shouldCreateJob(jobSummaries, "master") {
		t.Fatalf("Want false\n")
	}
}

func TestJobNotObsolete(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}
	jobSummary := jenkins.JobSummary{Branch: "origin/feature/1"}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobNotObsoleteBranchUnmanaged(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}
	jobSummary := jenkins.JobSummary{Branch: "origin/master"}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobObsolete(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}
	jobSummary := jenkins.JobSummary{Branch: "origin/feature/1"}
	stashBranches := map[string]stash.Branch{"feature/2": stash.Branch{}}
	if !s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}