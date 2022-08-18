#!/bin/bash
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
