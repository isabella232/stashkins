Stashkins
=========

Jenkins / Stash tooling

Stashkins queries a Jenkins server for all the jobs built against
a specified git repository URL and deletes jobs for which there is
no backing git branch and adds jobs to build branches it is not yet
building.  Stashkins currently operates against Atlassian Stash
only.  Stashkins will also create per-branch Maven repositories in
Sonatype Nexus, which works in concert with the job template to
determine to which Maven repository job artifacts should be published.

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
