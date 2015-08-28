package stashkins

import (
	"github.com/xoom/jenkins"
	"testing"
)

func TestCalculateMissingJobs(t *testing.T) {
	specCIJobs := []JobDescriptorNG{
		JobDescriptorNG{JobName: "name1"},
		JobDescriptorNG{JobName: "name2"},
	}

	jobSummaries := []jenkins.JobSummary{
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "name1"},
		},
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "name3"},
		},
	}

	missingJobs := DefaultStashkins{}.calculateMissingJobs(specCIJobs, jobSummaries)
	if len(missingJobs) != 1 {
		t.Fatalf("Want 1 but got %d\n", len(missingJobs))
	}
	if missingJobs[0].JobName != "name2" {
		t.Fatalf("Want name2 but got %s\n", missingJobs[0].JobName)
	}
}
