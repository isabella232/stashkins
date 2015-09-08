package maventools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var group string = `
{
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
}`

func TestGetRepoGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("Wanted GET but got %s\n", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service/local/repo_groups/snapshotgroup") {
			t.Fatalf("Wanted URL suffix /service/local/repo_groups/snapshotgroup but got: %s\n", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("Wanted application/json but got %s for Accept header", r.Header.Get("Accept"))
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatalf("Wanted an Authorization header but found none")
		}
		base64 := authHeader[len("Basic "):]
		if base64 != "dXNlcjpwYXNzd29yZA==" {
			t.Fatalf("Wanted dXNlcjpwYXNzd29yZA== but got %s\n", base64)
		}

		w.WriteHeader(200)
		fmt.Fprintf(w, "%s", group)
	}))
	defer server.Close()

	client := NewNexusClient(server.URL, "user", "password")
	group, rc, err := client.repositoryGroup("snapshotgroup")

	if err != nil {
		t.Fatalf("Expecting no error but got one: %v\n", err)
	}

	if rc != 200 {
		t.Fatalf("Want 200 but got %d\n", rc)
	}

	if group.Data.Provider != "maven2" {
		t.Fatalf("Want maven2 but got %s\n", group.Data.Provider)
	}
	if group.Data.Name != "SnapshotGroup" {
		t.Fatalf("Want SnapshotGroup but got %s\n", group.Data.Name)
	}
	if group.Data.Format != "maven2" {
		t.Fatalf("Want maven2 but got %s\n", group.Data.Format)
	}
	if group.Data.RepoType != "group" {
		t.Fatalf("Want group but got %s\n", group.Data.RepoType)
	}
	if group.Data.ID != "snapshotgroup" {
		t.Fatalf("Want snapshotgroup but got %s\n", group.Data.ID)
	}
	if !group.Data.Exposed {
		t.Fatalf("Want true but got false\n")
	}
	if group.Data.ContentResourceURI != "http://localhost:8081/nexus/content/groups/snapshotgroup" {
		t.Fatalf("Want http://localhost:8081/nexus/content/groups/snapshotgroup but got %s\n", group.Data.ContentResourceURI)
	}

	if len(group.Data.Repositories) != 1 {
		t.Fatalf("Wanted 1 repository but found %d\n", len(group.Data.Repositories))
	}
	repository := group.Data.Repositories[0]
	if repository.ID != "plat.trnk.trnk679" {
		t.Fatalf("Wanted plat.trnk.trnk679 but got %s\n", repository.ID)
	}
	if repository.Name != "plat.trnk.trnk679" {
		t.Fatalf("Wanted plat.trnk.trnk679 but got %s\n", repository.Name)
	}
	if repository.ResourceURI != "http://localhost:8081/nexus/service/local/repo_groups/snapshotgroup/plat.trnk.trnk679" {
		t.Fatalf("Wanted http://localhost:8081/nexus/service/local/repo_groups/snapshotgroup/plat.trnk.trnk679 but got %s\n", repository.ResourceURI)
	}
}

func TestGetRepoGroupNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewNexusClient(server.URL, "user", "password")
	_, rc, err := client.repositoryGroup("snapshotgroup")

	if err == nil {
		t.Fatalf("Expecting an error but got none\n")
	}

	if rc != 404 {
		t.Fatalf("Want 404 but got %d\n", rc)
	}
}
