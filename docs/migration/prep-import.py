# This script is part of the tuf-on-ci migration
#
# It takes "legacy" root-signing metadata (from repository/repository), writes
# copies in metadata/ in a format that is "compatible" with tuf-on-ci output.
# (in other words, changes whitespace rules and filepaths)
#
# The actions here do not affect the "canonical representation" of the metadata
# so to clients this metadata is the same as the legacy one.

import os
import shutil

from tuf.api.metadata import Metadata
from tuf.api.serialization.json import JSONSerializer

os.makedirs("metadata/root_history", exist_ok=True)

# Copy old root metadata as is: some of it is not considered valid by python-tuf
version = 1
while os.path.exists(f"repository/repository/{version}.root.json"):
    shutil.copyfile(f"repository/repository/{version}.root.json", f"metadata/root_history/{version}.root.json")
    version += 1

# current metadata, convert to tuf-on-ci layout
for rolename in ["root", "timestamp", "snapshot", "targets", "registry.npmjs.org"]:
    md: Metadata = Metadata.from_file(f"repository/repository/{rolename}.json")
    md.to_file(f"metadata/{rolename}.json", JSONSerializer())

# For the current root we do want the version in root_history to match root.json bit by bit
shutil.copyfile("metadata/root.json", f"metadata/root_history/{version - 1}.root.json")
