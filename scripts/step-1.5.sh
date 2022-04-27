#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi
# Online top-level keys
if [ -z "$TIMESTAMP_KEY" ]; then
    echo "Set TIMESTAMP_KEY"
    exit
fi
if [ -z "$SNAPSHOT_KEY" ]; then
    echo "Set SNAPSHOT_KEY"
    exit
fi
# Delegation keys
if [ -z "$REKOR_KEY" ]; then
    echo "Set REKOR_KEY"
    exit
fi
if [ -z "$STAGING_KEY" ]; then
    echo "Set STAGING_KEY"
    exit
fi
if [ -z "$REVOCATION_KEY" ]; then
    echo "Set REVOCATION_KEY"
    exit
fi
# Repo options
if [ -z "$PREV_REPO" ]; then
    echo "Set PREV_REPO"
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

# Copy the previous keys and repository into the new repository.
cp -r ${PREV_REPO}/* ${REPO}
mkdir -p ${REPO}/staged/targets

# Setup the root and targets
./tuf init -repository $REPO -target-meta config/targets-metadata.yml -snapshot ${SNAPSHOT_KEY} -timestamp ${TIMESTAMP_KEY} -previous "${PREV_REPO}"
# Add rekor delegation
cp targets/rekor.pub targets/rekor.0.pub
./tuf add-delegation -repository $REPO -name "rekor" -key $REKOR_KEY -path "rekor.*.pub" -target-meta config/rekor-metadata.yml
# Add staging project delegation
./tuf add-delegation -repository $REPO -name "staging" -key $STAGING_KEY -path "*"
# TODO: Add revoked project delegation
./tuf add-delegation -repository $REPO -name "revocation" -key $REVOCATION_KEY -path "*" -target-meta config/revocation-metadata.yml

git checkout -b setup-root
git add ceremony/
git commit -s -a -m "Setting up root for ${GITHUB_USER}"
git push -f origin setup-root

# Open the browser
open "https://github.com/${GITHUB_USER}/root-signing/pull/new/setup-root" || xdg-open "https://github.com/${GITHUB_USER}/root-signing/pull/new/setup-root"

