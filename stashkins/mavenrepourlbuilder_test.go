package stashkins

import "testing"

func TestMavenRepoUrlBuilder(t *testing.T) {
	wc := WebClientParams{URL: "http://localhost:9090/nexus"}

	s := MavenAspect{
		MavenRepositoryParams: MavenRepositoryParams{
			WebClientParams:       wc,
			PerBranchRepositoryID: "PerBranchID",
		},
	}

	want := "http://localhost:9090/nexus/content/repositories/PRJ.APP.feature_1"
	url := s.buildMavenRepositoryURL("PRJ", "APP", "feature/1")
	if url != want {
		t.Fatalf("Want %s but got %s\n", want, url)
	}

	want = "http://localhost:9090/nexus/content/repositories/snapshots"
	url = s.buildMavenRepositoryURL("PRJ", "APP", "develop")
	if url != "http://localhost:9090/nexus/content/repositories/snapshots" {
		t.Fatalf("Want %s but got %s\n", want, url)
	}
}
