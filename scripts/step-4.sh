#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
if [ -z "$TIMESTAMP_KEY" ]; then
    echo "Set TIMESTAMP_KEY"
    exit
fi
if [ -z "$SNAPSHOT_KEY" ]; then
    echo "Set SNAPSHOT_KEY"
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

# Snapshot and sign the snapshot with snapshot kms key
./tuf snapshot -repository $REPO
./tuf sign -repository $REPO -roles snapshot -key ${SNAPSHOT_KEY}

# Timestamp and sign the timestamp with timestamp kms key
./tuf timestamp -repository $REPO
./tuf sign -repository $REPO -roles timestamp -key ${TIMESTAMP_KEY}

if [ -n "$NO_PUSH" ]; then
    echo "Skipping push, exiting early..."
fi

git checkout -b sign-snapshot
git add ceremony/
git commit -s -a -m "Signing snapshot for ${GITHUB_USER}"
git push -f origin sign-snapshot

# Open the browser
export GITHUB_URL=$(git remote -v | awk '/^upstream/{print $2}'| head -1 | sed -Ee 's#(git@|git://)#https://#' -e 's@com:@com/@' -e 's#\.git$##')
export CHANGE_BRANCH=$(git symbolic-ref HEAD | cut -d"/" -f 3,4)
export PR_URL=${GITHUB_URL}"/compare/${BRANCH}..."${CHANGE_BRANCH}"?expand=1"
open "${PR_URL}" || xdg-open "${PR_URL}"
