#!/usr/bin/env bash
#
# Copyright 2022 The Sigstore Authors.
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

set -euo pipefail

# Gets the open snapshot/timestamp update pull requests of the repository
timestamp_update() {
    gh api -H "Accept: application/vnd.github.v3+json" "/repos/$GITHUB_REPOSITORY/pulls?head=sigstore:update-snapshot-timestamp" | jq '.[0]' | jq 'select (.!=null)'
}

UPDATE_PR=$(timestamp_update)
if [[ -z "$UPDATE_PR" ]]; then
    PULL_NUMBER=$(echo $UPDATE_PR | jq -r '.number')

    # Approve PR
    curl \
    -o review_output.json
    -X POST \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    https://api.github.com/repos/$GITHUB_REPOSITORY/pulls/$PULL_NUMBER/reviews

    REVIEW_ID=$(cat review_output.json | jq -r '.id')
    GH_TOKEN=$GITHUB_TOKEN gh api \
    --method POST \
    -H "Accept: application/vnd.github+json" \
    /repos/$GITHUB_REPOSITORY/pulls/$PULL_NUMBER/reviews/$REVIEW_ID/events \
    -f event='APPROVE'

    # Attempt to merge PR
    GH_TOKEN=$GITHUB_TOKEN gh api \
    --method PUT \
    -H "Accept: application/vnd.github+json" \
    /repos/$GITHUB_REPOSITORY/pulls/$PULL_NUMBER/merge \
    -f commit_title='Update Snapshot and Timestamp' \
    -f commit_message='update snapshot and timestamp' 
fi