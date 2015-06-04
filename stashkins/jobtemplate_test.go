package stashkins

import (
	"os"
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

func (f finfo) Size() int64 {
	return 0 // unused
}

func (f finfo) Mode() os.FileMode {
	return f.fileMode
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
