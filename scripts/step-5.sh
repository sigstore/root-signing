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

# Sign the root and targets
./tuf publish -repository $REPO
# Clear and copy into the repository/
rm -r repository/
cp -r $REPO/repository/. repository/

git checkout -b publish
git add ceremony/
git commit -s -a -m "Publishing for ${GITHUB_USER}!"
git push -f origin publish

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/publish" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/publish"
