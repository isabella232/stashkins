package stashkins

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xoom/jenkins"
)

func TestCloneTemplates(t *testing.T) {
	sourceRepoDirectory, err := extractTestTemplates()
	if err != nil {
		t.Fatalf("Unexpected error extracting source templates: %v\n", err)
	}

	cloneDirectory, err := ioutil.TempDir("", "git-")
	if err != nil {
		t.Fatalf("Unexpected error creating temp dir: %v\n", err)
	}

	templates, err := GetTemplates("file://"+sourceRepoDirectory, "master", cloneDirectory)
	if err != nil {
		os.RemoveAll(cloneDirectory)
		os.RemoveAll(sourceRepoDirectory)
		t.Fatalf("Unexpected error: %v\n", err)
	}

	if len(templates) != 2 {
		t.Fatalf("Want 2 but got %d\n", len(templates))
	}
	for _, tmpl := range templates {
		if tmpl.ProjectKey != "playg1" && tmpl.ProjectKey != "playg2" {
			t.Fatalf("Want playg1 or playg2 but got %s\n", tmpl.ProjectKey)
		}
		if tmpl.Slug != "microservice" && tmpl.Slug != "android" {
			t.Fatalf("Want microservice or android but got %s\n", tmpl.Slug)
		}
		if tmpl.Slug == "microservice" && tmpl.JobType != jenkins.Maven {
			t.Fatalf("Want maven type for microservice but got %s\n", tmpl.JobType)
		}
		if tmpl.Slug == "android" && tmpl.JobType != jenkins.Freestyle {
			t.Fatalf("Want freestyle type for android but got %s\n", tmpl.JobType)
		}
	}

	os.RemoveAll(cloneDirectory)
	os.RemoveAll(sourceRepoDirectory)
}

func extractTestTemplates() (string, error) {
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
