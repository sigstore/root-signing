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

set -o errexit
set -o nounset
set -o pipefail

# Gets the open snapshot/timestamp update pull requests of the repository
timestamp_update() {
    gh api -H "Accept: application/vnd.github.v3+json" "/repos/${GITHUB_REPOSITORY}/pulls?head=sigstore:update-snapshot-timestamp" | jq '.[0]' | jq 'select (.!=null)'
}

UPDATE_PR=$(timestamp_update)
if [[ -n "${UPDATE_PR}" ]]; then
    PULL_NUMBER=$(echo "${UPDATE_PR}" | jq -r '.number')
    echo "pull request found: "
    echo "${PULL_NUMBER}"

    # Approve PR
    curl \
    -o review_output.json \
    -X POST \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    https://api.github.com/repos/"${GITHUB_REPOSITORY}"/pulls/"${PULL_NUMBER}"/reviews

    echo "review: "
    cat review_output.json

    # TODO: Use gh pr review PR_NUMBER --approve
    REVIEW_ID=$(jq -r '.id' review_output.json)
    GH_TOKEN=$GITHUB_TOKEN gh api \
    --method POST \
    -H "Accept: application/vnd.github+json" \
    /repos/"${GITHUB_REPOSITORY}"/pulls/"${PULL_NUMBER}"/reviews/"${REVIEW_ID}"/events \
    -f event='APPROVE'

    # Attempt to merge PR
    GH_TOKEN="${GITHUB_TOKEN}" gh api \
    --method PUT \
    -H "Accept: application/vnd.github+json" \
    /repos/"${GITHUB_REPOSITORY}"/pulls/"${PULL_NUMBER}"/merge \
    -f commit_title='Update Snapshot and Timestamp' \
    -f commit_message='update snapshot and timestamp' \
    -f merge_methodstring='squash'

else
    echo "No open snapshot/timestamp pull request found"
fi
