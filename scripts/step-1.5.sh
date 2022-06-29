#!/bin/bash

# Print all commands and stop on errors
set -ex

source "./scripts/utils.sh"

# Check that a github user is set.
check_user

# Set REPO
set_repository

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

# Dump the git state and clean-up
print_git_state
clean_state

# Checkout the working branch
checkout_branch

# Copy the previous keys and repository into the new repository.
mkdir -p ${REPO}/staged/targets
cp -r ${PREV_REPO}/* ${REPO}
# Remove a key by ID that need to be removed from the root keyholders
if [[ -n $1 ]]; then 
    echo "Removing key: $1"
    rm -r ${REPO}/keys/$1
fi
# TODO(asraa): We need to copy up-to-date snapshot and timestamp from the published
# repository. Ideally we'd chain from `repository/repository`: see https://github.com/sigstore/root-signing/issues/288
cp repository/repository/{snapshot.json,timestamp.json} ${REPO}/repository

# Setup the root and targets
./tuf init -repository $REPO -target-meta config/targets-metadata.yml -snapshot ${SNAPSHOT_KEY} -timestamp ${TIMESTAMP_KEY} -previous "${PREV_REPO}"
# Add rekor delegation
./tuf add-delegation -repository $REPO -name "rekor" -key $REKOR_KEY -path "rekor.*.pub" -target-meta config/rekor-metadata.yml -terminating true
# Add staging project delegation
./tuf add-delegation -repository $REPO -name "staging" -key $STAGING_KEY -path "*"
# Add revoked project delegation
./tuf add-delegation -repository $REPO -name "revocation" -key $REVOCATION_KEY -path "*" -target-meta config/revocation-metadata.yml

commit_and_push_changes setup-root
