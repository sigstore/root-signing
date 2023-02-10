#!/usr/bin/env bash
#
# Copyright 2021 The Sigstore Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Print all commands and stop on errors
set -o errexit
set -o xtrace

# shellcheck source=./scripts/utils.sh
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
./tuf snapshot -repository "$REPO"
./tuf sign -repository "$REPO" -roles snapshot -key "${SNAPSHOT_KEY}"

# Timestamp and sign the timestamp with timestamp kms key
./tuf timestamp -repository "$REPO"
./tuf sign -repository "$REPO" -roles timestamp -key "${TIMESTAMP_KEY}"

commit_and_push_changes snapshot-timestamp
