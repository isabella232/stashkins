package stashkins

import "testing"

func TestCreateJob(t *testing.T) {
	// Template has a malformed token.
	err := DefaultStashkins{}.createJob([]byte("hello {{.Tag}"), "jobName", "")
	if err == nil {
		t.Fatal("Expecting a parse error on bad template")
	}
}
