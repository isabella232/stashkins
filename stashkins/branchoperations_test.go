package stashkins

import (
	"testing"
	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
)

func TestBranchIsManaged(t *testing.T) {
	s := BranchOperations{ManagedPrefixes:[]string{"feature/"}}

	if s.branchIsManaged("master") {
		t.Fatalf("want master managed == false but got true\n")
	}
	if !s.branchIsManaged("develop") {
		t.Fatalf("want develop managed == true but got true\n")
	}

	if !s.branchIsManaged("feature/somebranch") {
		t.Fatalf("want feature/somebranch managed == true but got false\n")
	}

	if !s.branchIsManaged("origin/feature/somebranch") {
		t.Fatalf("want origin/feature/somebranch managed == true but got false\n")
	}
}

func TestIsTargetBranch(t *testing.T) {
	s := BranchOperations{}
	jobSummary := jenkins.JobSummary{GitURL: "ssh://X"}
	if !s.isTargetJob(jobSummary, "ssh://X") {
		t.Fatalf("Want true\n")
	}
}

func TestIsNotTargetBranch(t *testing.T) {
	s := BranchOperations{}
	jobSummary := jenkins.JobSummary{GitURL: "ssh://X"}
	if s.isTargetJob(jobSummary, "ssh://XXX") {
		t.Fatalf("Want false\n")
	}
}

func TestIsTargetBranchHttpUrl(t *testing.T) {
	s := BranchOperations{}
	jobSummary := jenkins.JobSummary{GitURL: "http://X"}
	if s.isTargetJob(jobSummary, "ssh://XXX") {
		t.Fatalf("Want false\n")
	}
}

func TestIsFeatureBranch(t *testing.T) {
	s := BranchOperations{ManagedPrefixes:[]string{"feature/"}}

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
	s := BranchOperations{ManagedPrefixes:[]string{"feature/"}}
	jobSummaries := []jenkins.JobSummary{jenkins.JobSummary{Branch: "origin/feature/1"}}
	if !s.shouldCreateJob(jobSummaries, "feature/2") {
		t.Fatalf("Want true\n")
	}
}

func TestDoNotCreateJobAlreadyBeingBuilt(t *testing.T) {
	s := BranchOperations{}
	jobSummaries := []jenkins.JobSummary{jenkins.JobSummary{Branch: "origin/feature/1"}}
	if s.shouldCreateJob(jobSummaries, "feature/1") {
		t.Fatalf("Want false\n")
	}
}

func TestDoNotCreateJobForUnmanagedBranch(t *testing.T) {
	s := BranchOperations{}
	jobSummaries := make([]jenkins.JobSummary, 0)
	if s.shouldCreateJob(jobSummaries, "master") {
		t.Fatalf("Want false\n")
	}
}

func TestJobNotObsolete(t *testing.T) {
	s := BranchOperations{}
	jobSummary := jenkins.JobSummary{Branch: "origin/feature/1"}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobNotObsoleteBranchUnmanaged(t *testing.T) {
	s := BranchOperations{}
	jobSummary := jenkins.JobSummary{Branch: "origin/master"}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobObsolete(t *testing.T) {
	s := BranchOperations{ManagedPrefixes:[]string{"feature/"}}
	jobSummary := jenkins.JobSummary{Branch: "origin/feature/1"}
	stashBranches := map[string]stash.Branch{"feature/2": stash.Branch{}}
	if !s.shouldDeleteJob(jobSummary, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

