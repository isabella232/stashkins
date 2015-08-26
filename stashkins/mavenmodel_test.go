package stashkins_test

import (
	"reflect"
	"testing"

	"github.com/xoom/maventools"
	"github.com/xoom/stashkins/stashkins"
)

func TestMakeMavenModel(t *testing.T) {
	maven := stashkins.NewMavenAspect(
		stashkins.MavenRepositoryParams{WebClientParams: stashkins.WebClientParams{URL: "http://maven.example.com/nexus"}, FeatureBranchRepositoryGroupID: "repoId"},
		maventools.Client{},
		stashkins.BranchOperations{},
	)

	o := maven.MakeModel("jobName", "jobDescription", "http://example.com/dot.git", "feature/f", stashkins.JobTemplate{ProjectKey: "key", Slug: "slug"})
	switch str := o.(type) {
	case stashkins.MavenJob:
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

func TestMavenModelFields(t *testing.T) {
	model := stashkins.MavenJob{}
	r := reflect.ValueOf(&model)
	for _, v := range []string{"JobName", "Description", "BranchName", "RepositoryURL", "MavenSnapshotRepositoryURL", "MavenRepositoryID"} {
		if !reflect.Indirect(r).FieldByName(v).IsValid() {
			t.Fatalf("Want a field named %s\n", v)
		}
	}
}
