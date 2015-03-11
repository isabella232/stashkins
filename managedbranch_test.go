package main

import "testing"

func TestBranchIsManaged(t *testing.T) {
	if branchIsManaged("master") {
		t.Fatalf("want master managed == false but got true\n")
	}
	if !branchIsManaged("develop") {
		t.Fatalf("want develop managed == true but got true\n")
	}

	if !branchIsManaged("feature/somebranch") {
		t.Fatalf("want feature/somebranch managed == true but got false\n")
	}

	if !branchIsManaged("origin/feature/somebranch") {
		t.Fatalf("want origin/feature/somebranch managed == true but got false\n")
	}
}
