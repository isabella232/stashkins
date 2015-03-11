package main

import (
	"testing"

	"github.com/xoom/jenkins"
)

func TestCreateLackingJob(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{Name: "origin/feature/1"},
	}
	jobConfigs := []jenkins.JobConfig{
		jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}},
	}
	if !shouldCreateJob(jobConfigs, "feature/2") {
		t.Fatalf("Want true\n")
	}
}

func TestDoNotCreateJobAlreadyBeingBuilt(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{Name: "origin/feature/1"},
	}
	jobConfigs := []jenkins.JobConfig{
		jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}},
	}
	if shouldCreateJob(jobConfigs, "feature/1") {
		t.Fatalf("Want false\n")
	}
}

func TestDoNotConsiderJobBuildingMultipleBranches(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{Name: "origin/feature/1"},
		jenkins.Branch{Name: "origin/feature/2"},
	}
	jobConfigs := []jenkins.JobConfig{
		jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}},
	}
	if shouldCreateJob(jobConfigs, "doesntmatter") {
		t.Fatalf("Want false\n")
	}
}

func TestDoNotCreateJobForUnmanagedBranch(t *testing.T) {
	builtBranches := []jenkins.Branch{
		jenkins.Branch{},
	}
	jobConfigs := []jenkins.JobConfig{
		jenkins.JobConfig{SCM: jenkins.Scm{Branches: jenkins.Branches{Branch: builtBranches}}},
	}
	if shouldCreateJob(jobConfigs, "master") {
		t.Fatalf("Want false\n")
	}
}
