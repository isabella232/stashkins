package stashkins

import (
	"testing"

	"github.com/xoom/jenkins"
)

func TestCreateLackingJob(t *testing.T) {
	s := StatelessOperations{}
	jobSummaries := []jenkins.JobSummary{jenkins.JobSummary{Branch: "origin/feature/1"}}
	if !s.shouldCreateJob(jobSummaries, "feature/2") {
		t.Fatalf("Want true\n")
	}
}

func TestDoNotCreateJobAlreadyBeingBuilt(t *testing.T) {
	s := StatelessOperations{}
	jobSummaries := []jenkins.JobSummary{jenkins.JobSummary{Branch: "origin/feature/1"}}
	if s.shouldCreateJob(jobSummaries, "feature/1") {
		t.Fatalf("Want false\n")
	}
}

func TestDoNotCreateJobForUnmanagedBranch(t *testing.T) {
	s := StatelessOperations{}
	jobSummaries := make([]jenkins.JobSummary, 0)
	if s.shouldCreateJob(jobSummaries, "master") {
		t.Fatalf("Want false\n")
	}
}
