package stashkins

import "testing"

func TestMavenRepoID(t *testing.T) {
	o := StatelessOperations{}

	if s := o.mavenRepositoryID("INF", "test-me", "master"); s != "INF.test-me.master" {
		t.Fatalf("Want INF.test-me but got %s\n", s)
	}

	if s := o.mavenRepositoryID("INF", "test-me", "feature/888"); s != "INF.test-me.feature_888" {
		t.Fatalf("Want INF.test-me.feature_888 but got %s\n", s)
	}

	if s := o.mavenRepositoryID("INF", "test-me", "feature/888/part2"); s != "INF.test-me.feature_888_part2" {
		t.Fatalf("Want INF.test-me.feature_888_part2 but got %s\n", s)
	}
}
