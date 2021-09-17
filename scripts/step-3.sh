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
export REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')

# Dump the git state
git status
git remote -v

git clean -d -f
git checkout main
git pull upstream main
git status

# Snapshot and sign the snapshot with snapshot kms key
./tuf snapshot -repository $REPO
./tuf sign -repository $REPO -roles snapshot -key ${SNAPSHOT_KEY}

# Timestamp and sign the timestamp with timestamp kms key
./tuf timestamp -repository $REPO
./tuf sign -repository $REPO -roles timestamp -key ${TIMESTMAP_KEY}

git checkout -b sign-snapshot
git add ceremony/
git commit -s -a -m "Signing snapshot for ${GITHUB_USER}"
git push -f origin sign-snapshot

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-snapshot" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-snapshot"
