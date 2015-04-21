package stashkins

import "testing"

func TestMavenRepoID(t *testing.T) {
	wc := WebClientParams{URL: "http://localhost:9090/nexus"}

	o := MavenAspect{
		MavenRepositoryParams: MavenRepositoryParams{
			WebClientParams:       wc,
			PerBranchRepositoryID: "PerBranchID",
		},
	}

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
