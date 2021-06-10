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

# Add the key!
./tuf add-key -repository $REPO
git status
git checkout -b add-key
git add ceremony/
git commit -s -a -m "Adding initial key for ${GITHUB_USER}"
git push -f origin add-key

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/add-key"
