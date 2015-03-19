package stashkins

import "testing"

func TestBranchIsManaged(t *testing.T) {
	s := StatelessOperations{}

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
