package maventools

import (
	"testing"
)

func TestCanonicalize(t *testing.T) {
	g := RepositoryGroupData{ID: GroupID("theid"), Name: "thename", ContentResourceURI: "http://foo/bar", Repositories: make([]repository, 0)}
	r := repository{ID: RepositoryID("therepoid"), Name: "thereponame", ResourceURI: "http://repo"}
	g.Repositories = append(g.Repositories, r)

	c := canonicalize(repoGroup{Data: g})
	if c.ID != "theid" {
		t.Fatalf("Want theid but got %v\n", c.ID)
	}
	if c.Name != "thename" {
		t.Fatalf("Want thename but got %v\n", c.Name)
	}
	if c.ContentResourceURI != "http://foo/bar" {
		t.Fatalf("Want http://foo/bar but got %v\n", c.ContentResourceURI)
	}

	if len(c.Repositories) != 1 {
		t.Fatalf("Want 1 but got %v\n", len(c.Repositories))
	}

	repo := c.Repositories[0]
	if repo.ID != "therepoid" {
		t.Fatalf("Want therepoid but got %v\n", repo.ID)
	}
	if repo.Name != "thereponame" {
		t.Fatalf("Want thereponame but got %v\n", repo.Name)
	}
	if repo.ResourceURI != "http://repo" {
		t.Fatalf("Want http://repo but got %v\n", repo.ResourceURI)
	}
}
