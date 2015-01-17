Stashkins
=========

Jenkins / Stash tooling

Stashkins queries a Jenkins server for all the jobs built against
a specified git repository URL and deletes jobs for which there is
no backing git branch and adds jobs to build branches it is not yet
building.  Stashkins currently operates against Atlassian Stash only.

Stashkins does no write operations against Stash.  It only reads
from Stash to determine stale or missing Jenkins jobs.

Installation
============

     go get github.com/xoom/jenkins
     go get github.com/xoom/stash
     go test
     go build

Usage
=====

     $ ./stashkins -h
     Usage of ./stashkins-darwin-amd64:
       -do-artifactory=false: Do Maven repository management against JFrog Artifactory.  Precludes do-nexus.
       -do-nexus=false: Do Maven repository management against Sonatype Nexus.  Precludes do-artifactory.
       -jenkins-url="http://jenkins.example.com:8080": Jenkins Base URL
       -job-template-file="job-template.xml": Jenkins job template file.
       -maven-repo-base-url="http://localhost:8081/nexus": Maven repository management Base URL
       -maven-repo-password="": Password for Maven repository management
       -maven-repo-repository-groupID="": Repository groupID in which to group new per-branch repositories
       -maven-repo-username="": Username for Maven repository management
       -repository-project-key="": The Stash Project Key for the job-repository of interest.  For example, PLAYG.
       -repository-slug="": The Stash repository 'slug' for the job-repository of interest.  For example, 'trunk'.
       -stash-password="": Password for Stash authentication
       -stash-rest-base-url="http://stash.example.com:8080": Stash REST Base URL
       -stash-username="": Username for Stash authentication
       -version=false: Print build info from which stashkins was built

A sample Jenkins job template is provided in sample-job-template.xml.
It will like have to be modified for your needs, as will the
JobTemplate struct in stashkins.go that populates it.
