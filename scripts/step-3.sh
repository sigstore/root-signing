#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
if [ -z "$REKOR_KEY" ]; then
    echo "Set REKOR_KEY"
    exit
fi
if [ -z "$STAGING_KEY" ]; then
    echo "Set STAGING_KEY"
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

# Sign the delegations
./tuf sign -repository $REPO -roles rekor -key ${REKOR_KEY}
./tuf sign -repository $REPO -roles staging -key ${STAGING_KEY}

git checkout -b sign-delegations
git add ceremony/
git commit -s -a -m "Signing delegations for ${GITHUB_USER}"
git push -f origin sign-delegations

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-delegations" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-delegations"
