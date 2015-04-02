package stashkins

import (
	"testing"
)

func TestMakeMavenModel(t *testing.T) {
	webClientParams := WebClientParams{URL: "http://maven.example.com/nexus"}
	maven := MavenAspect{MavenRepositoryParams: MavenRepositoryParams{
		WebClientParams:       webClientParams,
		PerBranchRepositoryID: "repoId"},
	}
	o := maven.MakeModel("jobName", "jobDescription", "http://example.com/dot.git", "feature/f", Template{ProjectKey: "key", Slug: "slug"})
	switch str := o.(type) {
	case MavenJob:
		if str.JobName != "jobName" {
			t.Fatalf("Want jobName but got %s\n", str.JobName)
		}
		if str.Description != "jobDescription" {
			t.Fatalf("Want jobDescription but got %s\n", str.Description)
		}
		if str.BranchName != "feature/f" {
			t.Fatalf("Want feature/f but got %s\n", str.BranchName)
		}
		if str.RepositoryURL != "http://example.com/dot.git" {
			t.Fatalf("Want http://example.com/dot.git but got %s\n", str.RepositoryURL)
		}
		if str.MavenSnapshotRepositoryURL != "http://maven.example.com/nexus/content/repositories/key.slug.feature_f" {
			t.Fatalf("Want http://maven.example.com/nexus/content/repositories/key.slug.feature_f but got %s\n", str.MavenSnapshotRepositoryURL)
		}

	}
}
