package stashkins

import (
	"github.com/xoom/jenkins"
	"testing"
)

func TestCalculateObsoleteJobs(t *testing.T) {
	specCIJobs := []JobDescriptorNG{
		JobDescriptorNG{JobName: "proj-somelib-continuous-feature-issue-99"},
		JobDescriptorNG{JobName: "proj-somelib-continuous-feature-issue-100"},
	}

	jobSummaries := []jenkins.JobSummary{
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-continuous-feature-issue-99"},
		},
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-continuous-feature-issue-100"},
		},
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-continuous-feature-issue-101"},
		},
		jenkins.JobSummary{
			JobDescriptor: jenkins.JobDescriptor{Name: "proj-somelib-continuous-feature-issue-102"},
		},
	}

	skins := DefaultStashkins{}
	oldJobs := skins.calculateObsoleteCIJobs(specCIJobs, "proj", "somelib", jobSummaries)
	if len(oldJobs) != 2 {
		t.Fatalf("Want 2 but got %d\n", len(oldJobs))
	}

	for _, v := range oldJobs {
		jobName := v.JobName
		if !(jobName == "proj-somelib-continuous-feature-issue-101" || jobName == "proj-somelib-continuous-feature-issue-102") {
			t.Fatalf("Want proj-somelib-continuous-feature-issue-101 or proj-somelib-continuous-feature-issue-102 but got %s\n", jobName)
		}
	}
}
