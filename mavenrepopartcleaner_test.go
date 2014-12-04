package main

import "testing"

func TestRepoIDPartCleaner(t *testing.T) {
	if s := mavenRepoIDPartCleaner("foo/"); s != "foo_" {
		t.Fatalf("Want foo_ but got %s\n", s)
	}
	if s := mavenRepoIDPartCleaner("/foo/"); s != "_foo_" {
		t.Fatalf("Want _foo_ but got %s\n", s)
	}
	if s := mavenRepoIDPartCleaner("foo?"); s != "foo_" {
		t.Fatalf("Want foo_ but got %s\n", s)
	}
	if s := mavenRepoIDPartCleaner("foo&"); s != "foo_" {
		t.Fatalf("Want foo_ but got %s\n", s)
	}
}
