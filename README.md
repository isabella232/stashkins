Stashkins
=========

Jenkins / Stash tooling

The Jenkins Job Namespace
=========================

Stashkins is a tool to perform Jenkins job reconciliation.  This
means it reads the branches that exist on a given repository and
creates using a template Jenkins jobs to build each branch.  It
also deletes Jenkins jobs whose backing git branch has been removed.

Stashkins considers itself the owner of the Jenkins job namespace.
This means it will treat job names as indicators of not only whether
a job should be created to build a branch on a repository, but also
whether to delete a a job whose backing branch has been deleted.

For example, if a repository _bar_ in a Stash project _foo_ has a
branch _issue/1_, Stashkins will create a job named
foo-bar-continuous-issue-1 if no job with that name exists.  Job
names therefore encode project name, repository name, and branch
built.

When the backing branch _issue/1_ is deleted, Stashkins will observe
this job still exists and delete it because the job name starts
with the _job namespace_ _foo-bar-continous-_ without a backing
branch to provide the issue-1 job name suffix.

For Maven jobs, Stashkins will also create per-branch Maven
repositories in Sonatype Nexus, which works in concert with the job
template to determine to which Maven repository job artifacts should
be published.

Stashkins also supports Jenkins Freestyle projects.

Stashkins does no write operations against Stash.  It only reads
from Stash to determine stale or missing Jenkins jobs.

Build
=====

     make

which outputs binaries for Mac, Linux, Windows in the current working
directory.

Usage
=====

     $ ./stashkins-darwin-amd64 -h
     Usage of ./stashkins-darwin-amd64:
         -jenkins-base-url="http://jenkins.example.com:8080": Jenkins Base URL
         -job-template-repository-branch="master": Templates are held a Stash repository.  This is the branch from which to fetch the job template.
         -job-template-repository-url="": The Stash repository where job templates are stored..
         -maven-repo-base-url="http://localhost:8081/nexus": Maven repository management Base URL
         -maven-repo-password="": Password for Maven repository management user
         -maven-repo-repository-groupID="": Repository groupID in which to group new per-branch repositories
         -maven-repo-username="": User capable of doing automation of Maven repository management
         -password="": Password for automation user
         -stash-rest-base-url="http://stash.example.com:8080": Stash REST Base URL
         -username="": User capable of doing automation tasks on Stash and Jenkins
         -version=false: Print build info from which stashkins was built

Job templates are retrieved from a dedicated git repository denoted
by job-template-repository-url.  Stashkins will clone this repository
and walk the directory tree looking for project-key/slug/template.xml
files on which to base new jobs for project *project-key* and
repository *slug*.

Template Parameters Available to Users
======================================

The Jenkins job templates for new Maven jobs that Stashkins creates 
have available to them the following template parameters:

    JobName                    string // foo in ssh://git@example.com:9999/teamp/foo.git
    Description                string // mashup of repository URL and branch name.  This is used for the Jenkins job description.
    BranchName                 string // feature/PROJ-999, as in feature/PROJ-999
    RepositoryURL              string // The developer's software project's Git URL, as in ssh://git@example.com:9999/teamp/code.git
    MavenSnapshotRepositoryURL string // the Maven repository URL to which to publish this job's artifacts
