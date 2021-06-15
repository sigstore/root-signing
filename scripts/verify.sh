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

# Setup forks
git remote rm upstream || true
git remote add upstream git@github.com:sigstore/root-signing.git
git remote rm origin || true
git remote add origin git@github.com:"$GITHUB_USER"/root-signing.git
git remote -v

# build the verification binary
go build -o verify ./cmd/verify
[ -f piv-attestation-ca.pem ] || wget https://developers.yubico.com/PIV/Introduction/piv-attestation-ca.pem

# Fetch the pull request if specified and verify
if [[ ! -z "$1" ]]; then
    # Pull request to verify. If not supplied, use main
    echo "Pull Request: $1"
    git fetch upstream pull/$1/head:VERIFY
    git checkout VERIFY
fi

./verify --root piv-attestation-ca.pem --repository $REPO

# cleanup
git checkout main
git branch -D VERIFY || true

