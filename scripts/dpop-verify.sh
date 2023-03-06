#!/usr/bin/env bash
#
# Copyright 2023 The Sigstore Authors.
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
set -u

BRANCH=$(git rev-parse --abbrev-ref HEAD)
FORK_POINT=$(git merge-base --fork-point origin/main "${BRANCH}")
REPO=./repository
DELEGATION=$1
SIG_FILE="${REPO}"/staged/"${FORK_POINT}".sig

if [ ! -f "${SIG_FILE}" ]; then
    echo Expected signature file: "${SIG_FILE}" not found
    exit 1
fi

SIG=$(cat "${SIG_FILE}")

./tuf key-pop-verify \
      -role "${DELEGATION}" \
      -challenge "${DELEGATION}" \
      -nonce "${FORK_POINT}" \
      -repository "${REPO}" \
      -sig "${SIG}"
