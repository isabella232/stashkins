package nexus

import "testing"

func TestGroupMembership(t *testing.T) {
	ra := []repository{
		{ID: "foo", Name: "foo", ResourceURI: "blah"},
		{ID: "bar", Name: "bar", ResourceURI: "blah"},
	}
	group := repoGroup{Data: RepositoryGroupData{Repositories: ra}}

	present := repoIsInGroup("foo", group)
	if !present {
		t.Fatalf("Wanted true but got false\n")
	}

	absent := repoIsNotInGroup("baz", group)
	if !absent {
		t.Fatalf("Wanted true but got false\n")
	}

	removeRepo("foo", &group)
	if len(group.Data.Repositories) != 1 {
		t.Fatalf("Wanted 1 but got %d\n", len(group.Data.Repositories))
	}
	if repoIsInGroup("foo", group) {
		t.Fatalf("Wanted false but got true\n")
	}
	if repoIsNotInGroup("bar", group) {
		t.Fatalf("Wanted false but got true\n")
	}
}
