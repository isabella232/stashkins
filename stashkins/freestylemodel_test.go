package stashkins_test

import (
	"github.com/xoom/stashkins/stashkins"
	"reflect"
	"testing"
)

func TestFreestyleTasks(t *testing.T) {
	aspect := stashkins.NewFreestyleAspect()

	if err := aspect.PostJobCreateTasks("jobName", "jobDescription", "http://example.com/dot.git", "feature/f", stashkins.JobTemplate{}); err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}

	if err := aspect.PostJobDeleteTasks("jobName", "jobDescription", "http://example.com/dot.git", stashkins.JobTemplate{}); err != nil {
		t.Fatalf("Unexpected error: %v\n", err)
	}
}

func TestFreestyleModel(t *testing.T) {
	aspect := stashkins.NewFreestyleAspect()
	model := aspect.MakeModel("jobName", "jobDescription", "http://example.com/dot.git", "feature/f", stashkins.JobTemplate{})
	switch modelType := model.(type) {
	case stashkins.FreestyleJob:
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

func TestFreestyleModelFields(t *testing.T) {
	model := stashkins.FreestyleJob{}
	r := reflect.ValueOf(&model)
	for _, v := range []string{"JobName", "Description", "BranchName", "RepositoryURL"} {
		if !reflect.Indirect(r).FieldByName(v).IsValid() {
			t.Fatalf("Want a field named %s\n", v)
		}
	}
}
