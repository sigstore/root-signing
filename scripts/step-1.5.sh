#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
if [ -z "$TIMESTAMP_KEY" ]; then
    echo "Set TIMESTAMP_KEY"
    exit
fi
if [ -z "$SNAPSHOT_KEY" ]; then
    echo "Set SNAPSHOT_KEY"
    exit
fi
if [ -z "$REKOR_KEY" ]; then
    echo "Set REKOR_KEY"
    exit
fi
if [ -z "$STAGING_KEY" ]; then
    echo "Set STAGING_KEY"
    exit
fi
if [ -z "$PREV_REPO" ]; then
    echo "Set PREV_REPO"
    exit
fi
export REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')

# Copy the previous keys and repository into the new repository.
cp -r ${PREV_REPO}/* ${REPO}

# Dump the git state
git status
git remote -v

git clean -d -f
git checkout main
git pull upstream main
git status

# Setup the root and targets
./tuf init -repository $REPO -target targets/fulcio.crt.pem -target targets/fulcio_v1.crt.pem -target targets/rekor.pub -target targets/ctfe.pub -target targets/artifact.pub -snapshot ${SNAPSHOT_KEY} -timestamp ${TIMESTAMP_KEY} -previous ${PREV_REPO}
# Add rekor delegation
cp targets/rekor.pub targets/rekor.0.pub
./tuf add-delegation -repository $REPO -name "rekor" -key $REKOR_KEY -path "rekor.*.pub" -target targets/rekor.0.pub
# Add staging project delegation
./tuf add-delegation -repository $REPO -name "staging" -key $STAGING_KEY -path "*"

git checkout -b setup-root
git add ceremony/
git commit -s -a -m "Setting up root for ${GITHUB_USER}"
git push -f origin setup-root

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/setup-root" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/setup-root"

