# Existing Keyholder Responsibilities

This document outlines the responsibilities of an existing keyholder.

## Outline

Thank you for being a Sigstore Root Keyholder!

Make sure you have read over the keyholder responsibilities in [OVERVIEW.md](./OVERVIEW.md). This document will cover [testing](#testing) and the actual ceremony [signing](#ceremony-signing) to perform during the ceremony.

If you are testing with a new key, follow [New Signer Testing](./NEW_SIGNER.md/#testing).

## Testing

Complete the following steps in order:

### Pre-requisites

Ensure you have the following:
- [ ] A local Git installation and a Go development setup with support for the Go version [here](https://github.com/sigstore/root-signing/blob/1d4462a5deaffbe3055b5e3fe3c53d1918594159/go.mod#L3)
- [ ] SSH authentication for GitHub (see [here](https://docs.github.com/en/authentication/connecting-to-github-with-ssh))
- [ ] A USB port connection for your hardware key (beware of using a remote connection; the keyholder should not assume that magic occurs during an SSH session)
- [ ] A fresh environment. In particular, ensure that environment variables like `LOCAL`, `GITHUB_USER`, and `BRANCH` are unset before you begin.
- [ ] A fork of the [root-signing](https://github.com/sigstore/root-signing) repository. Click the "fork" button on GitHub and clone the forked repository.

### Setup 

- [ ] Test binary build: Set your `${GITHUB_USER}` with your GitHub username and execute the script:
```bash
export GITHUB_USER=${GITHUB_USER}
export LOCAL=1
./scripts/step-0.sh
```
This will setup your clone and build the TUF binary to use for metadata generation. This will also disable PR creations after each step and allow you to test changes locally.
 
### Metadata Creation

- [ ] Create new unsigned metadata with the following:
```bash
export TEST_KEY=./tests/test_data/cosign.key
export TIMESTAMP_KEY=$TEST_KEY
export SNAPSHOT_KEY=$TEST_KEY
export REKOR_KEY=$TEST_KEY
export STAGING_KEY=$TEST_KEY
export REVOCATION_KEY=$TEST_KEY
export PREV_REPO=$(pwd)/repository
./scripts/step-1.5.sh
```

- [ ] **CONFIRM** that you created a new directory under `ceremony/$DATE/staged/` and it is populated with a `root.json` and `targets.json` along with other delegation metadata files.

### Signing

- [ ] Sign the root and targets metadata with the following command. 
```
./scripts/step-2.sh
```

You will be prompted to insert your hardware key. Insert it and continue. Then, it will prompt you for your PIN twice to sign `root.json` and `targets.json`. This will populate a signature for your key id in the `signatures` section for these two top-level roles.

```
{
  "signatures": [
    {
      "keyid": "c2fbb0569e108fe928e6d6a55a5a18b646ebd8983ed9acc7a88446ef3955065f",
      "sig": "3044022046e1bb81175f2647751b142916a85fba3aad71162bfbe942b6b2cd2cbc2d5a3302205373a6e3f5a37f66a2bf7406315568734675b4b939795e98e4f292ad4e1a2e99"
    }
  ],
  [signed]
}
```

It will then prompt you to remove the hardware token. During the actual ceremony, it will push a PR to the root-signing repository.

- [ ] **CONFIRM**: Run the following and make sure that you see 1 valid signature for root and targets.
```bash
export REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')
./scripts/verify.sh
```

## Ceremony Signing

During the actual ceremony, you will need to renew the [setup](#setup-1) and perform a subset of the operations above. In particuar, you will only need to sign (you will not need to initialize the metadata yourself).

### Pre-requisites

Ensure you have the following:
- [ ] A local Git installation and a Go development setup with support for the Go version [here](https://github.com/sigstore/root-signing/blob/1d4462a5deaffbe3055b5e3fe3c53d1918594159/go.mod#L3)
- [ ] SSH authentication for GitHub (see [here](https://docs.github.com/en/authentication/connecting-to-github-with-ssh))
- [ ] A USB port connection for your hardware key (beware of using a remote connection; the keyholder should not assume that magic occurs during an SSH session)
- [ ] A fresh environment. In particular, ensure that environment variables like `LOCAL`, `GITHUB_USER`, and `BRANCH` are unset before you begin.

### Setup 

- [ ] Binary build: Set your `${GITHUB_USER}` with your GitHub username and execute the script:
```bash
export GITHUB_USER=${GITHUB_USER}
./scripts/step-0.sh
```
This will setup your clone and build the TUF binary to use for metadata generation.

### Signing

When prompted in the Slack channel, begin signing root and targets metadata. If the ceremony started on a prior date (or the date in your timezone does not match the date in the orchestrator's timezone), you should add a repository reference by setting `REPO`. Otherwise, for same-day signing, you may omit to default to the current date. Run:
```
REPO=$(pwd)/ceremony/<START_DATE> ./scripts/step-2.sh
```

Again, this will populate a signature for your key id in the `signatures` section for these two top-level roles and push a PR that will be verified and merged.
