package stashkins

import (
	"github.com/xoom/maventools/nexus"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMavenPostDeleteTasks(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := *r.URL
		if url.Path != "/service/local/repositories/PROJ.slug.feature_1" {
			t.Fatalf("postdelete URL path expected to be /nexus/service/local/repositories/PROJ.slug.feature_1 but found %s\n", url.Path)
		}
		if r.Header.Get("Authorization") != "Basic dTpw" {
			t.Fatalf("Want  Basic dTpw but found %s\n", r.Header.Get("Authorization"))
		}
		w.WriteHeader(204)
	}))
	defer testServer.Close()

	params := MavenRepositoryParams{
		PerBranchRepositoryID: "repoID",
	}
	nexusClient := nexus.NewClient(testServer.URL, "u", "p")

	aspect := MavenAspect{MavenRepositoryParams: params, Client: nexusClient}
	aspect.PostJobDeleteTasks("jobName", "ssh://git@example.com/dot.git", "origin/feature/1", Template{ProjectKey: "PROJ", Slug: "slug"})
}
