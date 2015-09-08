package stashkins

import (
	"testing"
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
	}

	if s.isFeatureBranch("origin/feature/*") {
		t.Fatalf("want false\n")
	}

	if s.isFeatureBranch("origin/feature/z") {
		t.Fatalf("want false\n")
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

	branchName, err := s.recoverBranchFromCIJobName("proj-slug-continuous-feature-PRJ-44")
	if err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}

	if branchName != "feature/PRJ-44" {
		t.Fatalf("Want feature/PRJ-44 but got %s\n", branchName)
	}
}

func TestFailedRecoverBranchNameFromCIJobName(t *testing.T) {
	s := BranchOperations{}

	_, err := s.recoverBranchFromCIJobName("blah")
	if err == nil {
		t.Fatal("Expected error for lacking -continuous- delimated in job name\n")
	}
}
