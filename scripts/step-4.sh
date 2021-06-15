#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
export REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')

# Dump the git state
git status
git remote -v

git clean -d -f
git checkout main
git pull upstream main
git status

# Sign the timestamp
./tuf sign -repository $REPO -roles timestamp

git checkout -b sign-timestamp
git add ceremony/
git commit -s -a -m "Signing timestamp for ${GITHUB_USER}"
git push -f origin sign-timestamp

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-timestamp" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-timestamp"
