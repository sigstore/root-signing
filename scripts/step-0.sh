#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
fi

if [ -z "$BRANCH" ]; then
    export BRANCH=main
else
    echo "Using branch $BRANCH"
fi

# Dump the git state
git status
git remote -v

# Setup forks
git remote rm upstream || true
git remote add upstream git@github.com:sigstore/root-signing.git
git remote rm origin || true
git remote add origin git@github.com:"$GITHUB_USER"/root-signing.git
git remote -v


# Cleanup branches
git branch -D setup-root || true
git branch -D add-key || true
git branch -D sign-root-targets || true
git branch -D sign-delegations || true
git branch -D sign-snapshot || true
git branch -D sign-timestamp || true
git branch -D publish || true

git clean -d -f
git checkout $BRANCH
git pull upstream $BRANCH
git rev-parse HEAD

# build the tuf binary
go build -o tuf -tags=pivkey ./cmd/tuf
