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

set -euo pipefail

#
# This is just a thin wrapper that takes on the input from a pull request
# and parses out the name of the delegation and the signature.
# It then calls the real script which will invoke the actual verification.
#
TITLE=$1

DELEGATION=$(echo "${TITLE}" | sed -E 's/(.+) for (.+)/\2/')
OUTPUT=$(mktemp)
./scripts/dpop-verify.sh "${DELEGATION}" 2>&1 | tee "${OUTPUT}"
