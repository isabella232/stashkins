package stashkins

import (
	"strings"

	"github.com/xoom/jenkins"
	"github.com/xoom/stash"
)

type BranchOperations struct {
	ManagedPrefixes []string
}

func NewBranchOperations(managedPrefixes string) BranchOperations {
	t := strings.Split(managedPrefixes, ",")
	prefixes := make([]string, 0)
	for _, v := range t {
		candidate := strings.TrimSpace(v)
		if candidate == "" {
			Log.Printf("In managed prefixes [%v], candidate has zero length.  Skipping.\n", managedPrefixes)
			continue
		}
		if !strings.HasSuffix(candidate, "/") {
			Log.Printf("In managed prefixes [%v], candidate '%s' is missing trailing /.  Skipping.", managedPrefixes, candidate)
			continue
		}
		prefixes = append(prefixes, candidate)
	}
	if len(prefixes) == 0 {
		Log.Printf("Managed branch prefixes length is zero.\n")
	}
	return BranchOperations{ManagedPrefixes: prefixes}
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

func (c BranchOperations) isBranchManaged(stashBranch string) bool {
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

func (c BranchOperations) shouldDeleteJob(jobSummary jenkins.JobSummary, stashBranches map[string]stash.Branch) bool {
	if !c.isBranchManaged(jobSummary.Branch) {
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
	if !c.isBranchManaged(branch) {
		return false
	}
	for _, jobSummary := range jobSummaries {
		if strings.HasSuffix(jobSummary.Branch, branch) {
			return false
		}
	}
	return true
}

func (c BranchOperations) stripLeadingOrigin(branch string) string {
	if strings.HasPrefix(branch, "origin/") {
		return branch[len("origin/"):]
	}
	return branch
}

func (c BranchOperations) recoverBranchFromCIJobName(jobName string) string {
	parts := strings.Split(jobName, "-continuous-")
	p := parts[1]
	return strings.Replace(p, "-", "/", 1)
}

func (c BranchOperations) canonicalCIJobName(projectKey, slug string, branch stash.Branch) string {
	// For a branch with Stash displayID feature/12, branchBaseName will be "feature" and branchSuffix will be "-12".
	// For a branch with Stash displayID develop, branchBaseName will be develop and branchSuffix will be an empty string.
	branchBaseName, branchSuffix := c.suffixer(branch.DisplayID)
	return projectKey + "-" + slug + "-continuous-" + branchBaseName + branchSuffix
}
