package stashkins

import (
	"fmt"
	"strings"
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

func (c BranchOperations) suffixer(branchDisplayID string) (string, string) {
	// For a branch with Stash displayID feature/12, branchBaseName will be "feature" and branchSuffix will be "-12".
	// For a branch with Stash displayID develop, branchBaseName will be develop and branchSuffix will be an empty string.
	s := strings.Split(branchDisplayID, "/")
	prefix := s[0]
	var suffix string

	if len(s) == 1 {
		return prefix, suffix
	}

	if len(s) == 2 {
		suffix = s[1]
	} else {
		suffix = branchDisplayID[strings.Index(branchDisplayID, "/")+1:]
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
		if strings.HasPrefix(branchName, managedPrefix) {
			return true
		}
	}
	return false
}

func (c BranchOperations) stripLeadingOrigin(branch string) string {
	if strings.HasPrefix(branch, "origin/") {
		return branch[len("origin/"):]
	}
	return branch
}

func (c BranchOperations) recoverBranchFromCIJobName(jobName string) (string, error) {
	parts := strings.Split(jobName, "-continuous-")
	if len(parts) != 2 {
		return "", fmt.Errorf("jobName %s split on -continuous- expected to have two parts.  Found %d\n", jobName, len(parts))
	}
	p := parts[1]
	return strings.Replace(p, "-", "/", 1), nil
}
