package stashkins

import (
	"testing"

	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
)

func TestNewBranchOperations(t *testing.T) {
	var branchOperations BranchOperations

	branchOperations = NewBranchOperations("feature/, hotfix/,,bug")
	if len(branchOperations.ManagedPrefixes) != 2 {
		t.Fatalf("Want 2 but got %d\n", len(branchOperations.ManagedPrefixes))
	}
	for _, v := range branchOperations.ManagedPrefixes {
		if !(v == "feature/" || v == "hotfix/") {
			t.Fatalf("Unexpected prefix %s\n", v)
		}
	}

	branchOperations = NewBranchOperations("feature/")
	if len(branchOperations.ManagedPrefixes) != 1 {
		t.Fatalf("Want 1 but got %d\n", len(branchOperations.ManagedPrefixes))
	}
	for _, v := range branchOperations.ManagedPrefixes {
		if v != "feature/" {
			t.Fatalf("Unexpected prefix %s\n", v)
		}
	}
}

func TestBranchIsManaged(t *testing.T) {
	s := BranchOperations{}
	if s.isBranchManaged("master") {
		t.Fatalf("want master managed == false but got true\n")
	}
	if !s.isBranchManaged("develop") {
		t.Fatalf("want develop managed == true but got true\n")
	}
}

func TestIsFeatureBranch(t *testing.T) {
	s := BranchOperations{ManagedPrefixes: []string{"feature/", "hotfix/"}}

	for _, prefix := range s.ManagedPrefixes {
		if !s.isBranchManaged(prefix + "somebranch") {
			t.Fatalf("want " + prefix + "somebranch managed == true but got false\n")
		}

		if !s.isBranchManaged("origin/" + prefix + "somebranch") {
			t.Fatalf("want origin/" + prefix + "somebranch managed == true but got false\n")
		}
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

func TestStripLeadingOrigin(t *testing.T) {
	s := BranchOperations{}

	var v string
	v = s.stripLeadingOrigin("origin/a_branch")
	if v != "a_branch" {
		t.Fatalf("Want a_branch but got %s\n", v)
	}

	v = s.stripLeadingOrigin("notorigin/a_branch")
	if v != "notorigin/a_branch" {
		t.Fatalf("Want notorigin/a_branch but got %s\n", v)
	}

	v = s.stripLeadingOrigin("develop")
	if v != "develop" {
		t.Fatalf("Want develop but got %s\n", v)
	}
}

func TestRecoverBranchNameFromCIJobName(t *testing.T) {
	s := BranchOperations{}
	branchName := s.recoverBranchFromCIJobName("proj-slug-continuous-feature-PRJ-44")
	if branchName != "feature/PRJ-44" {
		t.Fatalf("Want feature/PRJ-44 but got %s\n", branchName)
	}
}
