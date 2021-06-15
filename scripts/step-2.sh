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

# Sign the root and targets
./tuf sign -repository $REPO -roles root -roles targets

git checkout -b sign-targets
git add ceremony/
git commit -s -m "Signing targets for ${GITHUB_USER}"
git push -f origin sign-targets

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-targets" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-targets"
