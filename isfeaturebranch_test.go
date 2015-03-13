package main

import "testing"

func TestIsFeatureBranch(t *testing.T) {
	if isFeatureBranch("master") {
		t.Fatalf("Want false\n")
	}

	if isFeatureBranch("develop") {
		t.Fatalf("Want false\n")
	}

	if !isFeatureBranch("origin/feature/1") {
		t.Fatalf("Want true\n")
	}

	if !isFeatureBranch("feature/1") {
		t.Fatalf("Want true\n")
	}

    if isFeatureBranch("origin/feature/*") {
        t.Fatalf("want false\n")
    }
}
