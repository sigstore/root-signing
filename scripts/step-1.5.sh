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

# Setup the root
./tuf init -repository $REPO -target targets/fulcio.crt.pem -target targets/rekor.pub -target targets/ctfe.pub -target targets/artifact.pub

git checkout -b setup-root
git add ceremony/
git commit -s -a -m "Setting up root for ${GITHUB_USER}"
git push -f origin setup-root

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/setup-root" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/setup-root"

