package stashkins

type FreestyleAspect struct {
	Aspect
}

func NewFreestyleAspect() Aspect {
	return FreestyleAspect{}
}

func (fs FreestyleAspect) MakeModel(newJobName, newJobDescription, gitRepositoryURL, branch string, templateRecord Template) interface{} {
	return FreestyleJob{
		JobName:       newJobName,
		Description:   newJobDescription,
		BranchName:    branch,
		RepositoryURL: gitRepositoryURL,
	}
}

func (fs FreestyleAspect) PostJobCreateTasks(jobName, jobDescription, gitRepositoryURL, branch string, templateRecord Template) error {
	return nil
}

func (fs FreestyleAspect) PostJobDeleteTasks(jobName, jobDescription, gitRepositoryURL string, templateRecord Template) error {
	return nil
}
