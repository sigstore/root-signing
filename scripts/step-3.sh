#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$REPO" ]; then
    echo "Set REPO"
    exit
fi
if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi

# Dump the git state
git status
git remote -v

git clean -d -f
git checkout main
git pull upstream main
git status

# Sign the snapshot
./tuf sign -repository $REPO -roles snapshot

git checkout -b sign-snapshot
git commit -s -a -m "Signing snapshot for ${GITHUB_USER}"
git push -f origin sign-snapshot

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-snapshot"
