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

# Add the key!
./tuf add-key -repository $REPO
git status
git checkout -b add-key
git add ceremony/
git commit -s -a -m "Adding initial key for ${GITHUB_USER}"
git push -f origin add-key

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/add-key" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/add-key"
