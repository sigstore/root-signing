#!/bin/bash

# Print all commands and stop on errors
set -ex

source "./scripts/utils.sh"

# Check that a github user is set.
check_user

# Set REPO
set_repository

if [ -z "$TIMESTAMP_KEY" ]; then
    echo "Set TIMESTAMP_KEY"
    exit
fi
if [ -z "$SNAPSHOT_KEY" ]; then
    echo "Set SNAPSHOT_KEY"
    exit
fi

# Dump the git state and clean-up
print_git_state
clean_state

# Checkout the working branch
checkout_branch

# Snapshot and sign the snapshot with snapshot kms key
./tuf snapshot -repository $REPO
./tuf sign -repository $REPO -roles snapshot -key ${SNAPSHOT_KEY}

# Timestamp and sign the timestamp with timestamp kms key
./tuf timestamp -repository $REPO
./tuf sign -repository $REPO -roles timestamp -key ${TIMESTAMP_KEY}

commit_and_push_changes snapshot-timestamp
