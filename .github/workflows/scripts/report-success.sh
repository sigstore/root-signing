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

set -eo pipefail

source "./.github/workflows/scripts/e2e-utils.sh"

e2e_create_issue_success_body() {
    create_issue_body

    echo "" >>./BODY
    echo "**Tests are passing now. Closing this issue.**" >>./BODY

}

THIS_FILE=$(this_file)
e2e_create_issue_success_body

ISSUE_ID=$(gh -R "$ISSUE_REPOSITORY" issue list --label "bug" --state open -S "$THIS_FILE" --json number | jq '.[0]' | jq -r '.number' | jq 'select (.!=null)')

if [[ -n "$ISSUE_ID" ]]; then
    echo gh -R "$ISSUE_REPOSITORY" issue close "$ISSUE_ID" -c "$(cat ./BODY)"
    GH_TOKEN=$TOKEN gh -R "$ISSUE_REPOSITORY" issue close "$ISSUE_ID" -c "$(cat ./BODY)"
fi
