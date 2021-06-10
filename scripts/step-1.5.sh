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

# Setup the root
./tuf init -repository $REPO -target targets/fulcio.crt.pem

git checkout -b setup-root
git commit -s -a -m "Setting up root for ${GITHUB_USER}"
git push -f origin setup-root

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/setup-root"
