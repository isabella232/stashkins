package stashkins

import "testing"

func TestIsFeatureBranch(t *testing.T) {
	s := StatelessOperations{}

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
