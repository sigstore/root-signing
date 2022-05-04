#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
if [ -z "$REPO" ]; then
    REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')
    echo "Using default REPO $REPO"
fi

if [ -z "$BRANCH" ]; then
    export BRANCH=main
else
    echo "Using branch $BRANCH"
fi

# Dump the git state
git status
git remote -v

if [ -z "$NO_CLEAN" ]; then
    git clean -d -f
    git checkout $BRANCH
    git pull upstream $BRANCH
fi

git status

# Ask user to insert key 
read -n1 -r -s -p "Insert your Yubikey, then press any key to continue...\n" 

# Sign the root and targets with hardware key
./tuf sign -repository $REPO -roles root -roles targets -sk

# Ask user to remove key (and replace with SSH security key)
read -n1 -r -s -p "Remove your Yubikey, then press any key to continue...\n" 

if [ -n "$NO_PUSH" ]; then
    echo "Skipping push, exiting early..."
    exit
fi

git status
git checkout -b sign-root-targets
git add ceremony/
git commit -s -m "Signing root and targets for ${GITHUB_USER}"
git push -f origin sign-root-targets

# Open the browser
export GITHUB_URL=$(git remote -v | awk '/^upstream/{print $2}'| head -1 | sed -Ee 's#(git@|git://)#https://#' -e 's@com:@com/@' -e 's#\.git$##')
export CHANGE_BRANCH=$(git symbolic-ref HEAD | cut -d"/" -f 3,4)
export PR_URL=${GITHUB_URL}"/compare/${BRANCH}..."${CHANGE_BRANCH}"?expand=1"
open "${PR_URL}" || xdg-open "${PR_URL}"
