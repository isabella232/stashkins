package main

import (
	"testing"

	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
)

func TestJobNotObsolete(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{Name: "origin/feature/1"},
	}
	jobConfig := jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if shouldDeleteJob(jobConfig, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobNotObsoleteBranchUnmanaged(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{Name: "origin/master"},
	}
	jobConfig := jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}}
	stashBranches := map[string]stash.Branch{"feature/1": stash.Branch{}}
	if shouldDeleteJob(jobConfig, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobObsolete(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{Name: "origin/feature/1"},
	}
	jobConfig := jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}}
	stashBranches := map[string]stash.Branch{"feature/2": stash.Branch{}}
	if !shouldDeleteJob(jobConfig, stashBranches) {
		t.Fatalf("Want false\n")
	}
}

func TestJobBuildsMultipleBranches(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{},
		jenkins.Branch{},
	}
	jobConfig := jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}}
	stashBranches := map[string]stash.Branch{}
	if shouldDeleteJob(jobConfig, stashBranches) {
		t.Fatalf("Want false\n")
	}
}
