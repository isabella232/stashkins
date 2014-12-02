#!/bin/sh

set -ux

git push
git checkout master
git merge develop
git push
git checkout develop

