package main

import (
	"testing"

	"github.com/xoom/jenkins"
)

func TestIsTargetBranch(t *testing.T) {
	c := []jenkins.UserRemoteConfig{jenkins.UserRemoteConfig{URL: "ssh://X"}}
	remoteConfigs := jenkins.UserRemoteConfigs{
		UserRemoteConfig: c,
	}
	if !isTargetJob("foo", remoteConfigs, "ssh://X") {
		t.Fatalf("Want true but got false\n")
	}
}

func TestIsNotTargetBranch(t *testing.T) {
	c := []jenkins.UserRemoteConfig{jenkins.UserRemoteConfig{URL: "ssh://X"}}
	remoteConfigs := jenkins.UserRemoteConfigs{
		UserRemoteConfig: c,
	}
	if isTargetJob("foo", remoteConfigs, "ssh://XXX") {
		t.Fatalf("Want false but got true\n")
	}
}

func TestIsTargetBranchHttpUrl(t *testing.T) {
	c := []jenkins.UserRemoteConfig{jenkins.UserRemoteConfig{URL: "http://X"}}
	remoteConfigs := jenkins.UserRemoteConfigs{
		UserRemoteConfig: c,
	}
	if isTargetJob("foo", remoteConfigs, "ssh://XXX") {
		t.Fatalf("Want false but got true\n")
	}
}

func TestIsTargetBranchMultipleUserRemoteConfig(t *testing.T) {
	c := []jenkins.UserRemoteConfig{
		jenkins.UserRemoteConfig{},
		jenkins.UserRemoteConfig{},
	}
	remoteConfigs := jenkins.UserRemoteConfigs{
		UserRemoteConfig: c,
	}
	if isTargetJob("foo", remoteConfigs, "ssh://someurl") {
		t.Fatalf("Want false but got true\n")
	}
}
