package stashkins

import "testing"

func TestJobNameSpace(t *testing.T) {
	nameSpace := DefaultStashkins{}.jobNameSpace("proj", "somelib")
	if nameSpace != "proj-somelib-continuous-" {
		t.Fatalf("Want proj-somelib-continuous- but got %s\n", nameSpace)
	}
}

func TestJobIsInNameSpace(t *testing.T) {
	if !(DefaultStashkins{}.jobInNameSpace("proj-somelib-continuous-feature-99", "proj", "somelib")) {
		t.Fatalf("Expecting job to be in namespace\n")
	}

	if (DefaultStashkins{}.jobInNameSpace("proj-otherlib-continuous-feature-99", "proj", "somelib")) {
		t.Fatalf("Not expecting job to be in namespace\n")
	}
}
