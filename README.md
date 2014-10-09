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
