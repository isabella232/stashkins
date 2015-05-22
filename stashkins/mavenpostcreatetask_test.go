package stashkins_test

import (
	"fmt"
	"github.com/xoom/maventools/nexus"
	"github.com/xoom/stashkins/stashkins"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMavenPostCreateTasks(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Basic dTpw" {
			t.Fatalf("Want  Basic dTpw but found %s\n", r.Header.Get("Authorization"))
		}
		url := *r.URL
		switch url.String() {
		case "/service/local/repo_groups/repoID":
			if r.Method != "GET" && r.Method != "PUT" {
				t.Fatalf("Want GET or PUT for getting or mutating repo group got %s\n", r.Method)
			}
			fmt.Fprint(w, `{
   "data" : {
      "provider" : "maven2",
      "name" : "SnapshotGroup",
      "repositories" : [
         {
            "name" : "plat.trnk.trnk679",
            "id" : "plat.trnk.trnk679",
            "resourceURI" : "http://localhost:8081/nexus/service/local/repo_groups/snapshotgroup/plat.trnk.trnk679"
         }
      ],
      "format" : "maven2",
      "repoType" : "group",
      "exposed" : true,
      "id" : "snapshotgroup",
      "contentResourceURI" : "http://localhost:8081/nexus/content/groups/snapshotgroup"
   }
}`)
			return
		case "/service/local/repo_groups":
			if r.Method != "PUT" {
				t.Fatalf("Want PUT for adding repo to group but got %s\n", r.Method)
			}
			w.WriteHeader(204)
			return
		case "/service/local/repositories/PROJ.slug.feature_1":
			if r.Method != "GET" {
				t.Fatalf("Want GET for checking if repo exists but got %s\n", r.Method)
			}
			w.WriteHeader(200)
			return
		}
		t.Fatalf("Unexpected URL: %v\n", url)
	}))
	defer testServer.Close()

	params := stashkins.MavenRepositoryParams{
		FeatureBranchRepositoryGroupID: "repoID",
	}
	nexusClient := nexus.NewClient(testServer.URL, "u", "p")
	aspect := stashkins.NewMavenAspect(params, nexusClient, stashkins.BranchOperations{})
	aspect.PostJobCreateTasks("jobName", "jobDescription", "ssh://git@example.com/dot.git", "feature/1", stashkins.JobTemplate{ProjectKey: "PROJ", Slug: "slug"})
}
