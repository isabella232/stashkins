package maventools

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepoExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("Wanted GET but got %s\n", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/service/local/repositories/somerepo") {
			t.Fatalf("Wanted URL suffix /service/local/repositories/somerepo but got: %s\n", r.URL.Path)
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
	}))
	defer server.Close()

	var client NexusClient
	client = NewNexusClient(server.URL, "user", "password")
	exists, err := client.RepositoryExists("somerepo")
	if err != nil {
		t.Fatalf("Expecting no error but got one: %v\n", err)
	}

	if !exists {
		t.Fatalf("Wanted true but got false")
	}
}

func TestRepoNotExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewNexusClient(server.URL, "user", "password")
	exists, err := client.RepositoryExists("somerepo")
	if err != nil {
		t.Fatalf("Expecting no error but got one: %v\n", err)
	}

	if exists {
		t.Fatalf("Wanted false but got true")
	}
}

func TestRepoExistsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	client := NewNexusClient(server.URL, "user", "password")
	_, err := client.RepositoryExists("somerepo")
	if err == nil {
		t.Fatalf("Expecting error but got none\n")
	}
}
