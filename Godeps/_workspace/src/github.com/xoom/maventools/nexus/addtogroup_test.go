package nexus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddToGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(r.Method == "GET" || r.Method == "PUT") {
			t.Fatalf("Wanted GET or PUT but got %s\n", r.Method)
		}

		group := `{
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

		if r.Method == "GET" {
			fmt.Fprintf(w, "%s", group)
			return
		}

		if !strings.HasSuffix(r.URL.Path, "/service/local/repo_groups/agroup") {
			t.Fatalf("Wanted URL suffix /service/local/repo_groups/agroup but got: %s\n", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("Wanted application/json but got %s for Accept header", r.Header.Get("Accept"))
		}
		if r.Header.Get("Content-type") != "application/json" {
			t.Fatalf("Wanted application/json but got %s for Content-type header", r.Header.Get("Accept"))
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatalf("Wanted an Authorization header but found none")
		}
		base64 := authHeader[len("Basic "):]
		if base64 != "dXNlcjpwYXNzd29yZA==" {
			t.Fatalf("Wanted dXNlcjpwYXNzd29yZA== but got %s\n", base64)
		}

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Not expecting an error but got one: %v\n", err)
		}

		var repogroup repoGroup
		if err := json.Unmarshal(data, &repogroup); err != nil {
			t.Fatalf("Not expecting an error but got one: %v\n", err)
		}
		if len(repogroup.Data.Repositories) != 2 {
			t.Fatalf("Want 2 but got %d\n", len(repogroup.Data.Repositories))
		}
		if !repoIsInGroup("plat.trnk.trnk679", repogroup) {
			t.Fatalf("Not expecting true but got false\n")
		}
		if !repoIsInGroup("somerepo", repogroup) {
			t.Fatalf("Not expecting true but got false\n")
		}

		repository := repogroup.Data.Repositories[0]
		if repository.Name != "plat.trnk.trnk679" {
			t.Fatalf("Want plat.trnk.trnk679 but got %s\n", repository.Name)
		}
		if repository.ID != "plat.trnk.trnk679" {
			t.Fatalf("Want plat.trnk.trnk679 but got %s\n", repository.ID)
		}
		if repository.ResourceURI != "http://localhost:8081/nexus/service/local/repo_groups/snapshotgroup/plat.trnk.trnk679" {
			t.Fatalf("Want http://localhost:8081/nexus/service/local/repo_groups/snapshotgroup/plat.trnk.trnk679 but got %s\n", repository.ResourceURI)
		}

		w.WriteHeader(200)

	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "password")
	rc, err := client.AddRepositoryToGroup("somerepo", "agroup")
	if err != nil {
		t.Fatalf("Expecting no error but got one: %v\n", err)
	}

	if rc != 200 {
		t.Fatalf("Want 200 but got %d\n", rc)
	}
}
