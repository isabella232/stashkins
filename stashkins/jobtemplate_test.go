package stashkins

import (
	"archive/zip"
	"github.com/xoom/jenkins"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type finfo struct {
	name     string
	fileMode os.FileMode
}

func (f finfo) Name() string {
	return f.name
}

func (f finfo) Mode() os.FileMode {
	return f.fileMode
}

func (f finfo) Size() int64 {
	return 0 // unused
}

func (f finfo) ModTime() time.Time {
	return time.Now() // unused
}

func (f finfo) IsDir() bool {
	return false // unused
}

func (f finfo) Sys() interface{} {
	return "" // unused
}

func TestWalkerFunc(t *testing.T) {
	var buffer []string

	// should return the file
	buffer = make([]string, 0)
	templateWalker("template.xml", &buffer)("apath", finfo{name: "template.xml", fileMode: 0}, nil)
	if len(buffer) != 1 {
		t.Fatalf("Want 1 but got %d\n", len(buffer))
	}
	if buffer[0] != "apath" {
		t.Fatalf("Want apath but got %s\n", buffer[0])
	}

	// not a file
	buffer = make([]string, 0)
	templateWalker("template.xml", &buffer)("apath", finfo{name: "template.xml", fileMode: os.ModeSocket}, nil)
	if len(buffer) != 0 {
		t.Fatalf("Want 0 but got %d\n", len(buffer))
	}

	// no such file
	buffer = make([]string, 0)
	templateWalker("not-template.xml", &buffer)("apath", finfo{name: "template.xml", fileMode: 0}, nil)
	if len(buffer) != 0 {
		t.Fatalf("Want 0 but got %d\n", len(buffer))
	}
}

func TestTrackingKey(t *testing.T) {
	s := templateKey("a", "b", 1)
	if s != "a.b.1" {
		t.Fatalf("Want a.b.1 but got %s\n", s)
	}
}

func TestProjectCoordinates(t *testing.T) {
	proj, slug, err := projectCoordinates("/a/B/c.xml")
	if err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}
	if proj != "a" {
		t.Fatalf("Want a but got %s\n", proj)
	}
	if slug != "b" {
		t.Fatalf("Want b but got %s\n", slug)
	}

	proj, slug, err = projectCoordinates("a/b/c.xml")
	if err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}
	if proj != "a" {
		t.Fatalf("Want a but got %s\n", proj)
	}
	if slug != "b" {
		t.Fatalf("Want b but got %s\n", slug)
	}

	_, _, err = projectCoordinates("/a/b")
	if err == nil {
		t.Fatalf("Expected error for lack of file system parts\n")
	}

	_, _, err = projectCoordinates("/a/b/c")
	if err == nil {
		t.Fatalf("Expected error for file not ending in .xml\n")
	}
}

func TestBuildTemplates(t *testing.T) {
	sourceRepoDirectory, err := extractTestTemplates()
	if err != nil {
		t.Fatalf("Unexpected error extracting source templates: %v\n", err)
	}

	cloneDirectory, err := ioutil.TempDir("", "git-")
	if err != nil {
		t.Fatalf("Unexpected error creating temp dir: %v\n", err)
	}

	templates, err := Templates("file://"+sourceRepoDirectory, "master", cloneDirectory)
	if err != nil {
		os.RemoveAll(cloneDirectory)
		os.RemoveAll(sourceRepoDirectory)
		t.Fatalf("Unexpected error: %v\n", err)
	}

	if len(templates) != 2 {
		t.Fatalf("Want 2 but got %d\n", len(templates))
	}
	for _, template := range templates {
		if template.ProjectKey != "playg1" && template.ProjectKey != "playg2" {
			t.Fatalf("Want playg1 or playg2 but got %s\n", template.ProjectKey)
		}
		if template.Slug != "microservice" && template.Slug != "android" {
			t.Fatalf("Want microservice or android but got %s\n", template.Slug)
		}
		if template.Slug == "microservice" && template.JobType != jenkins.Maven {
			t.Fatalf("Want maven type for microservice but got %s\n", template.JobType)
		}
		if template.Slug == "android" && template.JobType != jenkins.Freestyle {
			t.Fatalf("Want freestyle type for android but got %s\n", template.JobType)
		}

		if template.ReleaseJobTemplate == nil {
			t.Fatalf("Expecting a non nil release job template: %#v\n", template)
		}
		if template.ContinuousJobTemplate == nil {
			t.Fatalf("Expecting a non nil continuous job template: %#v\n", template)
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
