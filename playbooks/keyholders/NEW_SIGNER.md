# New Keyholder Responsibilities

This document outlines the responsibilities of a new root keyholder.
Follow this document if you are a new holder!

## Outline

Congrats on becoming a new Sigstore Root Keyholder!

Make sure you have read over the keyholder responsibilities in [OVERVIEW.md](./OVERVIEW.md). This document will cover [testing](#testing) and the actual ceremony [signing](#ceremony-signing) to perform during the ceremony.

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
 
### Registration

Do not use an existing key that is already in-use and you need to continue using -- this process will wipe the key! 

- [ ] Add your key with the following.
```
./scripts/step-1.sh
```

This step will reset your hardware key and will set a PIN. Choose a PIN between 6 and 8 characters that you will remember for signing in later steps.

This will output three files (a public key, device certificate, and hardware certificate) in a directory named with your serial number `ceremony/YYYY-MM-DD/keys/${SERIAL_NUM}`. During the actual ceremony, it will push a PR to the root-signing repository.

- [ ] **CONFIRM** that you created a new directory under `ceremony/$DATE/keys/` with a new serial numbered `XXXXXX` directory. Run the following and confirm that there is some output with `VERIFIED KEY WITH SERIAL NUMBER XXXXXX`.
```bash
./scripts/verify.sh
```

**Troubleshooting**
1. You may need to install `libpcslite` to support hardware tokens. See [`go-piv`'s installation instructions for your platform.](https://github.com/go-piv/piv-go#installation).
2. If you hit the error
```
error: connecting to pscs: the Smart card resource manager is not running
```
then run the following to start the pcsc daemon (note: this may require root access):
```
systemctl start pcscd.service
systemctl enable pcscd.service
```

### Metadata Creation

- [ ] Create new unsigned metadata with the following:
```bash
export TEST_KEY=./tests/test_data/cosign.key
export TIMESTAMP_KEY=$TEST_KEY
export SNAPSHOT_KEY=$TEST_KEY
export REKOR_KEY=$TEST_KEY
export STAGING_KEY=$TEST_KEY
export REVOCATION_KEY=$TEST_KEY
export PREV_REPO=$(pwd)/ceremony/2022-07-12
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

During the actual ceremony, you will need to renew the [setup](#setup-1) and perform a subset of the operations above. In particuar, you will only need to register and sign (you will not need to initialize the metadata yourself).

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

### Registration

The first step of the ceremony will require new keyholders to register their keys. Wait for a Slack message asking you to start Registration, and run:
```
./scripts/step-1.sh
```

Like testing, this will create a new directory under `ceremony/$DATE/keys/` with a new serial numbered `XXXXXX` directory. It will also push a PR that the community will verify and merge.

### Signing

When prompted in the Slack channel, begin signing root and targets metadata. Run:
```
./scripts/step-2.sh
```

Again, this will populate a signature for your key id in the `signatures` section for these two top-level roles and push a PR that will be verified and merged.
