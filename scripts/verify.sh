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
# set -o xtrace

# shellcheck source=./scripts/utils.sh
source "./scripts/utils.sh"

# Check that a github user is set.
check_user

# Set REPO
set_repository

# Dump the git state and clean-up
print_git_state
clean_state

# Setup forks
setup_forks

# Checkout branch
checkout_branch

# build the verification binary
go build -o verify ./cmd/verify
[ -f piv-attestation-ca.pem ] || curl -fsO https://developers.yubico.com/PIV/Introduction/piv-attestation-ca.pem

# Fetch the pull request if specified and verify
if [[ -n "$1" ]]; then
    # Pull request to verify. If not supplied, use main
    echo "Pull Request: $1"
    git branch -D VERIFY || true
    git fetch upstream pull/"$1"/head:VERIFY
    git checkout VERIFY
fi

echo
echo "Enter the number for the option you would like to verify:"
echo -e "\t 1: Verify the HSM keys and serial numbers in $REPO/keys"
echo -e "\t 2: Verify the signatures on any staged metadata"
echo -e "\t 3: Verify published data and targets in $REPO/repository"
read input

case $input in
    1)
      # Verify keys if keys/ repository exists.
        if [ -d "$REPO"/keys ]; then
            ./verify keys --root piv-attestation-ca.pem --key-directory "$REPO"/keys
        else
            echo "Error: Missing $REPO/keys sudirectory" && exit 1
        fi
        ;;
    2)
        # If staged metadata exists, verify the staged repository
        if [ -f "$REPO"/staged/root.json ]; then
            ./verify repository --repository "$REPO" --staged
        else
            echo "Error: Missing $REPO/staged subdirectory" && exit 1
        fi
        ;;
    3)
        echo "Enter comma-separated target names to verify. If blank, all top-level targets will be verified:"
        read targets
        echo "no " ${targets:+--targets $targets}
        echo "no col" ${targets+--targets $targets}

        # If published data exists, verify against a root
        if [ -f "$REPO"/repository/1.root.json ]; then
            ./verify repository --repository "$REPO" --root "$REPO"/repository/1.root.json ${targets:+--targets $targets}
        else
            echo "Error: Missing valid $REPO/repository TUF repository" && exit 1
        fi
        ;;
    *)
        echo "Error: Invalid user option" && exit 1;;
esac

# Stay on the branch for manual verification
