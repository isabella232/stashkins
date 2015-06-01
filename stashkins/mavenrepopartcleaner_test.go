package stashkins

import "testing"

func TestRepoIDPartCleaner(t *testing.T) {
	aspect := MavenAspect{}

	var out string

	out = aspect.scrubRepositoryID("abc?&//")
	if out != "abc____" {
		t.Fatalf("Want abc____ but got %s\n", out)
	}

	out = aspect.scrubRepositoryID("123abc_.-")
	if out != "123abc_.-" {
		t.Fatalf("Want 123abc_.- but got %s\n", out)
	}
}
