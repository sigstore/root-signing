#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
if [ -z "$CEREMONY_DATE" ]; then
    CEREMONY_DATE=$(date '+%Y-%m-%d')
fi
export REPO=$(pwd)/ceremony/$CEREMONY_DATE

# Dump the git state
git status
git remote -v

git clean -d -f
git checkout main
git pull upstream main
git status

# Sign the root and targets with hardware key
./tuf sign -repository $REPO -roles root -roles targets -sk

git checkout -b sign-targets
git add ceremony/
git commit -s -m "Signing root and targets for ${GITHUB_USER}"
git push -f origin sign-root-targets

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-root-targets" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/sign-root-targets"
