package stashkins

import (
	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
	"strings"
)

type BranchOperations struct {
	ManagedPrefixes []string
}

func (c BranchOperations) suffixer(branch string) (string, string) {
	s := strings.Split(branch, "/")
	prefix := s[0]
	var suffix string

	if len(s) == 1 {
		return prefix, suffix
	}

	if len(s) == 2 {
		suffix = s[1]
	} else {
		suffix = branch[strings.Index(branch, "/")+1:]
		suffix = strings.Replace(suffix, "/", "-", -1)
	}
	return prefix, "-" + suffix
}

func (c BranchOperations) branchIsManaged(stashBranch string) bool {
	return stashBranch == "develop" || c.isFeatureBranch(stashBranch)
}

func (c BranchOperations) isFeatureBranch(branchName string) bool {
	if strings.Contains(branchName, "*") {
		return false
	}
	for _, managedPrefix := range c.ManagedPrefixes {
		if strings.Contains(branchName, managedPrefix) {
			return true
		}
	}
	return false
}

func (c BranchOperations) isTargetJob(jobSummary jenkins.JobSummary, jobRepositoryURL string) bool {
	return jobSummary.GitURL == jobRepositoryURL
}

func (c BranchOperations) shouldDeleteJob(jobSummary jenkins.JobSummary, stashBranches map[string]stash.Branch) bool {
	if !c.branchIsManaged(jobSummary.Branch) {
		return false
	}
	deleteJobConfig := true
	for stashBranch, _ := range stashBranches {
		if strings.HasSuffix(jobSummary.Branch, stashBranch) {
			deleteJobConfig = false
		}
	}
	return deleteJobConfig
}

func (c BranchOperations) shouldCreateJob(jobSummaries []jenkins.JobSummary, branch string) bool {
	if !c.branchIsManaged(branch) {
		return false
	}
	for _, jobSummary := range jobSummaries {
		if strings.HasSuffix(jobSummary.Branch, branch) {
			return false
		}
	}
	return true
}
