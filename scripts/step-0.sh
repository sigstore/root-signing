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

# Dump the git state
print_git_state

# Setup forks
setup_forks

# Cleanup branches
cleanup_branches
clean_state

# Checkout the working branch
checkout_branch

# build the tuf binary
go build -o tuf -tags=pivkey ./cmd/tuf
