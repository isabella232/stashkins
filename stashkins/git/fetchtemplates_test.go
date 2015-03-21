package git

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchTemplates(t *testing.T) {
	sourceRepoDirectory, err := extractTemplates()
	if err != nil {
		t.Fatalf("Unexpected error extracting source templates: %v\n", err)
	}

	cloneDirectory, err := ioutil.TempDir("", "git-")
	if err != nil {
		t.Fatalf("Unexpected error creating temp dir: %v\n", err)
	}

	err = FetchTemplates("file://"+sourceRepoDirectory, "master", cloneDirectory)
	if err != nil {
		os.RemoveAll(cloneDirectory)
		os.RemoveAll(sourceRepoDirectory)
		t.Fatalf("Unexpected error: %v\n", err)
	}

	err = FetchTemplates("file://"+sourceRepoDirectory, "master", cloneDirectory)
	if err != nil {
		os.RemoveAll(cloneDirectory)
		os.RemoveAll(sourceRepoDirectory)
		t.Fatalf("Unexpected error: %v\n", err)
	}

	os.RemoveAll(cloneDirectory)
	os.RemoveAll(sourceRepoDirectory)
}

func extractTemplates() (string, error) {
	r, err := zip.OpenReader("test-templates-git.zip")
	if err != nil {
		return "", err
	}
	defer r.Close()

	name, err := ioutil.TempDir("", "templates-")
	if err != nil {
		return "", err
	}

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}

		destinationFileName := name + "/" + f.Name
		parentDir := filepath.Dir(destinationFileName)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return "", err
		}

		dst, err := os.Create(destinationFileName)
		if err != nil {
			return "", err
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		_, err = io.Copy(dst, rc)
		if err != nil {
			return "", err
		}
		rc.Close()
		dst.Close()
	}
	return name, nil
}
