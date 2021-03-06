package stashkins

import (
	"os"
	"os/exec"
)

// Clone the repository with native git and checkout the given branch to the given directory.
func cloneTemplates(repositoryURL, branch, dir string) error {
	if exists, err := dirExists(dir + "/.git"); err == nil && !exists {
		return clone(repositoryURL, branch, dir)
	} else {
		if err != nil {
			return err
		}
	}
	return pull(dir)
}

func clone(repositoryURL, branch, dir string) error {
	return executeShellCommand("git", []string{"clone", "--branch", branch, repositoryURL, dir})
}

func pull(dir string) error {
	err := os.Chdir(dir)
	if err != nil {
		return err
	}
	return executeShellCommand("git", []string{"pull"})
}

func executeShellCommand(commandName string, args []string) error {
	Log.Printf("Executing %s %+v\n", commandName, args)
	command := exec.Command(commandName, args...)
	var stdOutErr []byte
	var err error
	stdOutErr, err = command.CombinedOutput()
	if err != nil {
		return err
	}
	Log.Printf("%v\n", string(stdOutErr))

	return nil
}

func dirExists(dirPath string) (bool, error) {
	if _, err := os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
