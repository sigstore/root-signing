#!/bin/bash

# Print all commands and stop on errors
set -ex

source "./scripts/utils.sh"

# Check that a github user is set.
check_user

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

# Set REPO
set_repository

# Dump the git state and clean-up
print_git_state
clean_state

# Checkout the working branch
checkout_branch

# Sign the delegations
./tuf sign -repository $REPO -roles rekor -key ${REKOR_KEY}
./tuf sign -repository $REPO -roles staging -key ${STAGING_KEY}
./tuf sign -repository $REPO -roles revocation -key ${REVOCATION_KEY}

commit_and_push_changes sign-delegations
