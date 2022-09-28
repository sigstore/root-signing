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

# shellcheck source=./scripts/utils.sh
source "./scripts/utils.sh"

# Check that a github user is set.
check_user

# Set REPO
set_repository

# Dump the git state and clean-up
print_git_state
clean_state

# Checkout the working branch
checkout_branch

# Ask user to insert key
read -n1 -r -s -p "Insert your Yubikey, then press any key to continue...\n"

# Sign the root and targets with hardware key
# TODO(https://github.com/sigstore/root-signing/issues/381):
# Adding the explicit deprecated flag can be removed after v5 root-signing
./tuf sign -repository "$REPO" -roles root -roles targets -sk -add-deprecated true

# Ask user to remove key (and replace with SSH security key)
read -n1 -r -s -p "Remove your Yubikey, then press any key to continue...\n"

commit_and_push_changes sign-root-targets
