package nexus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeleteRepo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("Wanted DELETE but got %s\n", r.Method)
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
		w.WriteHeader(204)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "password")
	i, err := client.DeleteRepository("somerepo")
	if err != nil {
		t.Fatalf("Expecting no error but got one: %v\n", err)
	}
	if i != 204 {
		t.Fatalf("Want 204 but got %d\n", i)
	}
}

func TestDeleteRepoNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "password")
	i, err := client.DeleteRepository("somerepo")
	if err != nil {
		t.Fatalf("Expecting no error but got one: %v\n", err)
	}
	if i != 404 {
		t.Fatalf("Want 404 but got %d\n", i)
	}
}

func TestDeleteRepoUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "password")
	i, err := client.DeleteRepository("somerepo")
	if err == nil {
		t.Fatalf("Expecting an error but got none\n")
	}
	if i != 401 {
		t.Fatalf("Want 401 but got %d\n", i)
	}
}
