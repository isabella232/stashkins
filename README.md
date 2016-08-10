Stashkins
=========

Jenkins / Stash tooling

The Jenkins Job Namespace
=========================

Stashkins is a tool that performs Jenkins job reconciliation.  This
means it reads the branches that exist on a given repository and
uses a template to create jobs to build each one.  It also deletes
Jenkins jobs whose backing git branch has been removed.

Stashkins considers itself the owner of the Jenkins job namespace.
This means it will treat job names as indicators of not only whether
a job should be created to build a branch on a repository, but also
whether to delete a job whose backing branch has been deleted.

For example, if a repository _bar_ in a Stash project _foo_ has a
branch _issue/1_, Stashkins will create a job named
foo-bar-continuous-issue-1 if no job with that name exists.  Job
names therefore encode project name, repository name, and branch
built.

When the backing branch _issue/1_ is deleted, Stashkins will observe
this job still exists and delete it because the job name starts
with the _job namespace_ _foo-bar-continous-_ without a backing
branch to provide the suffix of the branch-part of the job name.

For Maven jobs, Stashkins will also create per-branch Maven
repositories in Sonatype Nexus, which works in concert with the job
template to determine to which Maven repository job artifacts should
be published.

Stashkins also supports Jenkins Freestyle projects.

Stashkins does no write operations against Stash.  It only reads
from Stash to determine stale or missing Jenkins jobs.

Build
=====

Stashkins uses [glide](https://github.com/Masterminds/glide) to manage dependencies.

The Makefile does a check to see if vendor/ exists.  If it does,
it assumes glide install has been run.  It's up to you to make sure
vendor/ is up to date before you go build.

```
$ glide install
$ make
```

which outputs binaries for Mac, Linux, Windows in the current working
directory.

Usage
=====

```
laptop:stashkins> ./stashkins-darwin-amd64 -h
Usage of ./stashkins-darwin-amd64:
  -jenkins-base-url string
    	Jenkins Base URL (default "http://jenkins.example.com:8080")
  -jenkins-jobs-directory string
    	Filesystem location of Jenkins jobs directory.  Used when acquiring job summaries from the Jenkins master filesystem.
  -job-template-repository-branch string
    	Templates are held a Stash repository.  This is the branch from which to fetch the job template. (default "master")
  -job-template-repository-url string
    	The Stash repository where job templates are stored..
  -managed-branch-prefixes string
    	Branch prefixes to manage. (default "feature/")
  -maven-repo-base-url string
    	Maven repository management Base URL (default "http://localhost:8081/nexus")
  -maven-repo-password string
    	Password for Maven repository management user
  -maven-repo-repository-groupID string
    	Repository groupID in which to group new per-branch repositories
  -maven-repo-username string
    	User capable of doing automation of Maven repository management
  -password string
    	Password for automation user
  -stash-rest-base-url string
    	Stash REST Base URL (default "http://stash.example.com:8080")
  -username string
    	User capable of doing automation tasks on Stash and Jenkins
  -version
    	Print build info from which stashkins was built
```

Job templates are retrieved from a dedicated git repository denoted
by _job-template-repository-url_.  Stashkins will clone this
repository and walk the directory tree looking for
project-key/slug/continous-template.xml and
project-key/slug/release-template.xml files on which to base new
CI and release jobs, respectively, for project *project-key* and
repository *slug*.

If _jenkins-job-directory_ is set, Stashkins will retrieve job
summaries from the filesystem on the Jenkins master.  If omitted,
job summaries will be retrieved over HTTP from the Jenkins master
URL.  Retrieving summaries from the filesystem can be tens of times
faster than over HTTP, especially when the number of jobs is large.

Template Parameters Available to Users
======================================

The Jenkins job templates for new Maven jobs that Stashkins creates 
have available to them the following template parameters:

    JobName                    string // foo in ssh://git@example.com:9999/teamp/foo.git
    Description                string // mashup of repository URL and branch name.  This is used for the Jenkins job description.
    BranchName                 string // feature/PROJ-999, as in feature/PROJ-999
    RepositoryURL              string // The developer's software project's Git URL, as in ssh://git@example.com:9999/teamp/code.git
    MavenSnapshotRepositoryURL string // the Maven repository URL to which to publish this job's artifacts
