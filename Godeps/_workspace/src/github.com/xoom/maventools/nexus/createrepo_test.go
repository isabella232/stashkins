package nexus

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateRepo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("Wanted POST but got %s\n", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service/local/repositories") {
			t.Fatalf("Wanted URL suffix /service/local/repositories but got: %s\n", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("Wanted application/json but got %s for Accept header", r.Header.Get("Accept"))
		}
		if r.Header.Get("Content-type") != "application/xml" {
			t.Fatalf("Wanted application/xml but got %s for Content-type header", r.Header.Get("Content-type"))
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatalf("Wanted an Authorization header but found none")
		}
		base64 := authHeader[len("Basic "):]
		if base64 != "dXNlcjpwYXNzd29yZA==" {
			t.Fatalf("Wanted dXNlcjpwYXNzd29yZA== but got %s\n", base64)
		}

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Got an error but was not expecting one: %v\n", err)
		}

		var repo createrepo
		err = xml.Unmarshal(b, &repo)
		if err != nil {
			t.Fatalf("Not expecting an error but got one: %v\n", err)
		}

		if repo.Data.Id != "somerepo" {
			t.Fatalf("Want somerepo but got %v\n", repo.Data.Id)
		}
		if repo.Data.Name != "somerepo" {
			t.Fatalf("Want somerepo but got %v\n", repo.Data.Name)
		}
		if repo.Data.Provider != "maven2" {
			t.Fatalf("Want maven2 but got %v\n", repo.Data.Provider)
		}
		if repo.Data.RepoType != "hosted" {
			t.Fatalf("Want hosted but got %v\n", repo.Data.RepoType)
		}
		if repo.Data.RepoPolicy != "SNAPSHOT" {
			t.Fatalf("Want SNAPSHOT but got %v\n", repo.Data.RepoPolicy)
		}
		if repo.Data.ProviderRole != "org.sonatype.nexus.proxy.repository.Repository" {
			t.Fatalf("Want org.sonatype.nexus.proxy.repository.Repository but got %v\n", repo.Data.ProviderRole)
		}
		if !strings.HasSuffix(repo.Data.ContentResourceURI, "/content/repositories/somerepo") {
			t.Fatalf("Want suffix /content/repositories/somerepo but got %v\n", repo.Data.ContentResourceURI)
		}
		if repo.Data.Format != "maven2" {
			t.Fatalf("Want maven2 but got %v\n", repo.Data.Format)
		}
		if !repo.Data.Exposed {
			t.Fatalf("Want true but got false\n")
		}

		w.WriteHeader(201)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "password")
	i, err := client.CreateSnapshotRepository("somerepo")
	if err != nil {
		t.Fatalf("Expecting no error but got one: %v\n", err)
	}
	if i != 201 {
		t.Fatalf("Want 201 but got %d\n", i)
	}
}

func TestCreateRepoWithError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "password")
	i, err := client.CreateSnapshotRepository("somerepo")
	if err == nil {
		t.Fatalf("Expecting an error but did not get one\n")
	}
	if i != 400 {
		t.Fatalf("Want 400 but got %d\n", i)
	}
}
