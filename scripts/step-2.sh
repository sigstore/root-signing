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

# Sign the root and targets
./tuf sign -repository $REPO -roles root -roles targets

git checkout -b sign-targets
git commit -s -a -m "Signing targets for ${GITHUB_USER}"
git push -f origin sign-targets

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-targets"
