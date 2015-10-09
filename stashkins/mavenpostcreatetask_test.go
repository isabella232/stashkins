package stashkins_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"fmt"
	"github.com/xoom/maventools"
	"github.com/xoom/stashkins/stashkins"
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
			if r.Method != "HEAD" {
				t.Fatalf("Want HEAD for checking if repo exists but got %s\n", r.Method)
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
	nexusClient := maventools.NewNexusClient(testServer.URL, "u", "p")
	aspect := stashkins.NewMavenAspect(params, nexusClient, stashkins.BranchOperations{ManagedPrefixes: []string{"feature/"}})
	err := aspect.PostJobCreateTasks("jobName", "jobDescription", "ssh://git@example.com/dot.git", "feature/1", stashkins.JobTemplate{ProjectKey: "PROJ", Slug: "slug"})
	if err != nil {
		t.Fatalf("Unexpected error: %n\n", err)
	}
}

func TestMavenPostCreateTasksCreatedRepoUnsettled(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Basic dTpw" {
			t.Fatalf("Want  Basic dTpw but found %s\n", r.Header.Get("Authorization"))
		}
		url := *r.URL
		switch url.Path {
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
			if r.Method != "HEAD" {
				t.Fatalf("Want HEAD for checking if repo exists but got %s\n", r.Method)
			}
			w.WriteHeader(404)  // this should cause the post-creator to error out waiting forever for the repo-create to settle with 200 OK
			return
		case "/service/local/repositories":
			if r.Method != "POST" {
				t.Fatalf("Want POST for creating new repository but got %s\n", r.Method)
			}
			w.WriteHeader(201)
			return
		}

		t.Fatalf("Unexpected URL: %v\n", url.Path)
	}))
	defer testServer.Close()

	params := stashkins.MavenRepositoryParams{
		FeatureBranchRepositoryGroupID: "repoID",
	}
	nexusClient := maventools.NewNexusClient(testServer.URL, "u", "p")
	aspect := stashkins.NewMavenAspect(params, nexusClient, stashkins.BranchOperations{ManagedPrefixes: []string{"feature/"}})
	err := aspect.PostJobCreateTasks("jobName", "jobDescription", "ssh://git@example.com/dot.git", "feature/1", stashkins.JobTemplate{ProjectKey: "PROJ", Slug: "slug"})
	if err == nil {
		t.Fatal("Expecting an error while verifying repo-create has a settled repository")
	}
}
