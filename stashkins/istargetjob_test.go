package stashkins

import (
	"testing"

	"github.com/xoom/jenkins"
)

func TestIsTargetBranch(t *testing.T) {
	s := StatelessOperations{}
	jobSummary := jenkins.JobSummary{GitURL: "ssh://X"}
	if !s.isTargetJob(jobSummary, "ssh://X") {
		t.Fatalf("Want true\n")
	}
}

func TestIsNotTargetBranch(t *testing.T) {
	s := StatelessOperations{}
	jobSummary := jenkins.JobSummary{GitURL: "ssh://X"}
	if s.isTargetJob(jobSummary, "ssh://XXX") {
		t.Fatalf("Want false\n")
	}
}

func TestIsTargetBranchHttpUrl(t *testing.T) {
	s := StatelessOperations{}
	jobSummary := jenkins.JobSummary{GitURL: "http://X"}
	if s.isTargetJob(jobSummary, "ssh://XXX") {
		t.Fatalf("Want false\n")
	}
}
