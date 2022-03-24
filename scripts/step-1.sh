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

# Ask user to insert key 
read -n1 -r -s -p "Insert your Yubikey, then press any key to continue..." 

# Add the key!
./tuf add-key -repository $REPO

# Ask user to remove key (and replace with SSH security key)
read -n1 -r -s -p "Remove your Yubikey, then press any key to continue..." 

git status
git checkout -b add-key
git add ceremony/
git commit -s -a -m "Adding initial key for ${GITHUB_USER}"
git push -f origin add-key

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/add-key" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/add-key"
