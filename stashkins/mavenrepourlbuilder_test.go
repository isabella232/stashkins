package stashkins

import "testing"

func TestMavenRepoUrlBuilder(t *testing.T) {
	wc := WebClientParams{URL: "http://localhost:9090/nexus"}

	s := MavenAspect{
		mavenRepositoryParams: MavenRepositoryParams{
			WebClientParams:                wc,
			FeatureBranchRepositoryGroupID: "PerBranchID",
		},
	}

	want := "http://localhost:9090/nexus/content/repositories/PRJ.APP.feature_1"
	url := s.repositoryURL("PRJ", "APP", "feature/1")
	if url != want {
		t.Fatalf("Want %s but got %s\n", want, url)
	}

	want = "http://localhost:9090/nexus/content/repositories/snapshots"
	url = s.repositoryURL("PRJ", "APP", "develop")
	if url != "http://localhost:9090/nexus/content/repositories/snapshots" {
		t.Fatalf("Want %s but got %s\n", want, url)
	}
}
