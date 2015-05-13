package stashkins

import "testing"

func TestFreestyleTasks(t *testing.T) {
	aspect := NewFreestyleAspect()

	if err := aspect.PostJobCreateTasks("jobName", "jobDescription", "http://example.com/dot.git", "feature/f", Template{}); err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}

	if err := aspect.PostJobDeleteTasks("jobName", "jobDescription", "http://example.com/dot.git", Template{}); err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}
}

func TestFreestyleModel(t *testing.T) {
	aspect := NewFreestyleAspect()
	model := aspect.MakeModel("jobName", "jobDescription", "http://example.com/dot.git", "feature/f", Template{})
	switch modelType := model.(type) {
	case FreestyleJob:
		if modelType.JobName != "jobName" {
			t.Fatalf("Want jobName but got %s\n", modelType.JobName)
		}
		if modelType.Description != "jobDescription" {
			t.Fatalf("Want jobDescription but got %s\n", modelType.Description)
		}
		if modelType.BranchName != "feature/f" {
			t.Fatalf("Want feature/f but got %s\n", modelType.BranchName)
		}
		if modelType.RepositoryURL != "http://example.com/dot.git" {
			t.Fatalf("Want http://example.com/dot.git but got %s\n", modelType.RepositoryURL)
		}
		return
	}
	t.Fatalf("Want FreestyleModel type\n")
}
