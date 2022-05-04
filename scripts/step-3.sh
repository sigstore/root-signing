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
if [ -z "$REVOCATION_KEY" ]; then
    echo "Set REVOCATION_KEY"
    exit
fi
if [ -z "$REPO" ]; then
    REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')
    echo "Using default REPO $REPO"
fi

if [ -z "$BRANCH" ]; then
    export BRANCH=main
else
    echo "Using branch $BRANCH"
fi

# Dump the git state
git status
git remote -v

git clean -d -f
git checkout $BRANCH
git pull upstream $BRANCH
git status

# Sign the delegations
./tuf sign -repository $REPO -roles rekor -key ${REKOR_KEY}
./tuf sign -repository $REPO -roles staging -key ${STAGING_KEY}

if [ -n "$NO_PUSH" ]; then
    echo "Skipping push, exiting early..."
fi

git checkout -b sign-delegations
git add ceremony/
git commit -s -a -m "Signing delegations for ${GITHUB_USER}"
git push -f origin sign-delegations

# Open the browser
export GITHUB_URL=$(git remote -v | awk '/^upstream/{print $2}'| head -1 | sed -Ee 's#(git@|git://)#https://#' -e 's@com:@com/@' -e 's#\.git$##')
export CHANGE_BRANCH=$(git symbolic-ref HEAD | cut -d"/" -f 3,4)
export PR_URL=${GITHUB_URL}"/compare/${BRANCH}..."${CHANGE_BRANCH}"?expand=1"
open "${PR_URL}" || xdg-open "${PR_URL}"
