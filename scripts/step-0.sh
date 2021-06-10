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
git remote rm upstream
git remote add upstream git@github.com:sigstore/root-signing.git
git remote rm origin
git remote add origin git@github.com:"$GITHUB_USER"/root-signing.git
git remote -v

git clean -d -f
git checkout main
git pull upstream main
git rev-parse HEAD

# build the tuf binary
go build -o tuf ./cmd/tuf
