#!/bin/bash

# Print all commands and stop on errors
set -ex

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit
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
git branch -D sign-targets || true
git branch -D sign-snapshot || true
git branch -D sign-timestamp || true
git branch -D publish || true

git clean -d -f
git checkout main
git pull upstream main
git rev-parse HEAD

# build the tuf binary
go build -o tuf -tags=pivkey ./cmd/tuf
