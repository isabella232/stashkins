package main

import "testing"

func TestSuffixer(t *testing.T) {
	var a, b string
	a, b = suffixer("a/b")
	if a != "a" {
		t.Fatalf("wanted a but found %s\n", a)
	}
	if b != "-b" {
		t.Fatalf("wanted -b but found %s\n", b)
	}

	a, b = suffixer("a/b/c")
	if a != "a" {
		t.Fatalf("wanted a but found %s\n", a)
	}
	if b != "-b-c" {
		t.Fatalf("wanted -b-c but found %s\n", b)
	}

	a, b = suffixer("a/b/c/d")
	if a != "a" {
		t.Fatalf("wanted a but found %s\n", a)
	}
	if b != "-b-c-d" {
		t.Fatalf("wanted -b-c-d but found %s\n", b)
	}

	a, b = suffixer("develop")
	if a != "develop" {
		t.Fatalf("wanted develop but found %s\n", a)
	}
	if b != "" {
		t.Fatalf("wanted empty string but found %s\n", b)
	}
}
