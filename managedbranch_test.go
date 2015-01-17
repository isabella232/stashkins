package main

import "testing"

func TestBranchIsManaged(t *testing.T) {
	if branchIsManaged("master") {
		t.Fatalf("want false but got true\n")
	}
	if branchIsManaged("develop") {
		t.Fatalf("want false but got true\n")
	}

	if !branchIsManaged("feature/somebranch") {
		t.Fatalf("want true but got false\n")
	}
	if !branchIsManaged("origin/feature/somebranch") {
		t.Fatalf("want true but got false\n")
	}
}
