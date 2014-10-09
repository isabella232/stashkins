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
     Usage of ./stashkins:
       -jenkins-url="http://jenkins.example.com:8080": Jenkins Base URL
       -job-repository-url="ssh://git@example.com:9999/teamp/code.git": The Git repository URL referenced by the Jenkins jobs.
       -job-sync=false: Sync Jenkins state against Stash for a given Stash repository.  Requires -job-repository-url.
       -job-template-file="job-template.xml": Jenkins job template file.
       -stash-password="": Password for Stash authentication
       -stash-rest-base-url="http://stash.example.com:8080": Stash REST Base URL
       -stash-username="": Username for Stash authentication

A sample Jenkins job template is provided in sample-job-template.xml.
It will like have to be modified for your needs, as will the
JobTemplate struct in stashkins.go that populates it.
