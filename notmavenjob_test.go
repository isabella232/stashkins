package main

import (
	"bytes"
	"encoding/xml"
	"github.com/xoom/jenkins"
	"testing"
)

// A highly truncated non maven2-moduleset job to study how to handle a specific type of XML unmarshal failure.
var config = `
<?xml version='1.0' encoding='UTF-8'?>
<project>
  <actions/>
  <description>Job builder job</description>
  <logRotator class="hudson.tasks.LogRotator">
    <daysToKeep>1</daysToKeep>
    <numToKeep>-1</numToKeep>
    <artifactDaysToKeep>-1</artifactDaysToKeep>
    <artifactNumToKeep>-1</artifactNumToKeep>
  </logRotator>
</project>
`

func TestNotMaven(t *testing.T) {
	var jobConfig jenkins.JobConfig
	reader := bytes.NewBuffer([]byte(config))
	if err := xml.NewDecoder(reader).Decode(&jobConfig); err != nil {
		if !isIgnoreable(err) {
			t.Fatalf("Wanted isIgnoreable==true\n")
		}
	}
}
