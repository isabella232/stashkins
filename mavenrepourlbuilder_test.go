package main

import "testing"

func TestMavenRepoUrlBuilder(t *testing.T) {
	want := "http://localhost:9090/nexus/content/repositories/PRJ.APP.feature_1"
	url := buildMavenRepositoryURL("http://localhost:9090/nexus", "PRJ", "APP", "feature/1")
	if url != want {
		t.Fatalf("Want %s but got %s\n", want, url)
	}

	want = "http://localhost:9090/nexus/content/repositories/snapshots"
	url = buildMavenRepositoryURL("http://localhost:9090/nexus", "PRJ", "APP", "develop")
	if url != "http://localhost:9090/nexus/content/repositories/snapshots" {
		t.Fatalf("Want %s but got %s\n", want, url)
	}
}
