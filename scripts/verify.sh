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

if [ -z "$GITHUB_USER" ]; then
    echo "Set GITHUB_USER"
    exit 1
fi
if [ -z "$REPO" ]; then
    REPO=$(pwd)/repository
    echo "Using default REPO $REPO"
fi

# Dump the git state
git checkout main
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
[ -f piv-attestation-ca.pem ] || curl -fsO https://developers.yubico.com/PIV/Introduction/piv-attestation-ca.pem

# Fetch the pull request if specified and verify
if [[ -n "$1" ]]; then
    # Pull request to verify. If not supplied, use main
    echo "Pull Request: $1"
    git branch -D VERIFY || true
    git fetch upstream pull/"$1"/head:VERIFY
    git checkout VERIFY
fi

# Verify keys if keys/ repository exists. It does not in the top-level published repository/
if [ -d "$REPO"/keys ]; then
    ./verify keys --root piv-attestation-ca.pem --key-directory "$REPO"/keys
fi
# If staged metadata exists, verify the staged repository
if [ -f "$REPO"/staged/root.json ]; then
    ./verify repository --repository "$REPO" --staged
fi
# If published data exists, verify against a root
if [ -f "$REPO"/repository/1.root.json ]; then
    ./verify repository --repository "$REPO" --root "$REPO"/repository/1.root.json
fi
# stay on the branch for manual verification
