package stashkins

import (
	"github.com/xoom/jenkins"
	"testing"
)

func TestShouldCreateReleaseJob(t *testing.T) {
	skins := DefaultStashkins{}
	if !skins.shouldCreateReleaseJob("proj", "somelib", []jenkins.JobSummary{
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-continuous-feature-issue-99"},
		},
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-continuous-feature-issue-100"},
		},
	}) {
		t.Fatalf("Want true when creating release job\n")
	}

	if skins.shouldCreateReleaseJob("proj", "somelib", []jenkins.JobSummary{
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-continuous-feature-issue-99"},
		},
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-release"},
		},
	}) {
		t.Fatalf("Want false when creating release job\n")
	}

}
