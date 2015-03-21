package git

import (
	"io/ioutil"
	"os"
	"os/user"
	"testing"
)

func TestDirExists(t *testing.T) {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Cannot get current user: %v\n", err)
	}

	exists, err := dirExists(currentUser.HomeDir)
	if err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}

	if !exists {
		t.Fatalf("Want true\n")
	}
}

func TestDirNotExists(t *testing.T) {
	name, err := ioutil.TempDir("", "tmp-")
	if err != nil {
		t.Fatalf("Cannot create temp dir.  Unexpected error: %v\n", err)
	}
	defer os.RemoveAll(name)

	exists, err := dirExists(name + "/foo")
	if err != nil {
		os.RemoveAll(name)
		t.Fatalf("Unexpected error: %v\n", err)
	}

	if exists {
		os.RemoveAll(name)
		t.Fatalf("Want true\n")
	}
}
