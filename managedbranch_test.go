package main

import "testing"

func testBranchIsManaged(t *testing.T) {
	if !branchIsManaged("master") {
		t.Fatalf("want false but got true\n")
	}
	if !branchIsManaged("develop") {
		t.Fatalf("want false but got true\n")
	}
	if branchIsManaged("feature/somebranch") {
		t.Fatalf("want true but got false\n")
	}
}
