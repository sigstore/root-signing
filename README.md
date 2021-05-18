# test-sigstore-root

This repository contains the SigStore TUF metadata and the scripts needed to generate and verify it. 

The repository metadata will be located in `repository/metadata.staged` and the targets in `repository/targets`.
It will be populated using the target files contained in `targets/` and the keys in the `ceremony/<YYYY-MM-DD>/keys`.
