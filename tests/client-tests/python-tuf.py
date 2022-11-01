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

import os
import shutil
import sys
import tempfile

from tuf.ngclient import Updater

REPO_URL = os.getenv("REPO")

with tempfile.TemporaryDirectory() as tmpdirname:
    METADATA_DIR = f"{tmpdirname}/metadata"
    os.mkdir(METADATA_DIR)

    # Copy in root metadata to use as trusted root.
    # NOTE: we have to use v5 or newer because prior versions were not
    # compatible with python-tuf:
    # https://github.com/sigstore/root-signing/issues/103
    # https://github.com/sigstore/root-signing/issues/329
    shutil.copyfile(
        "repository/repository/5.root.json",
        f"{METADATA_DIR}/root.json")

    # TODO: we hard-code a single target here and will need to update this
    # if the target retrieval API changes.
    fulcio_cert = "fulcio.crt.pem"
    try:
        updater = Updater(
            metadata_dir=METADATA_DIR,
            metadata_base_url=f"{REPO_URL}",
            target_base_url=f"{REPO_URL}/targets/",
            target_dir=tmpdirname)

        info = updater.get_targetinfo(fulcio_cert)

        if info is None:
            print(f"Failed to fetch {fulcio_cert}")
            sys.exit(1)

        path = updater.download_target(info)
        print(f"Fetched {fulcio_cert} to {path}")

    except Exception as e:
        print(f"Updated and fetch of {fulcio_cert} failed")
        print(e)
        sys.exit(2)
