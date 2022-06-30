# Keyholder Responsibilities

This document outlines the responsibilities of a root keyholder.

## Outline

Keyholders MUST subscribe to the [Sigstore Maintainer Calendar](https://calendar.google.com/calendar/u/0?cid=Y19ydjIxcDJuMzJsbmJoYW5uaXFwOXIzNTJtb0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t) for upcoming root signing event. Root signing events are expected to occur about every 4-5 months. The next `v+1` version signing will be scheduled, and the `v+2` version will be tentatively scheduled. Testing will occur the week before the signing. Keyholders are required to communicate that they have completed the [pre-work](../KEYHOLDER.md#signing-pre-work) to the orchestrator through [#sigstore-keyholder](https://sigstore.slack.com/archives/C03E4HP6RCK) Slack channel. All testing and signing events can occur asynchronously. Keyholders are expected to be "on-call" during the root signing window in case there is an issue.

### Pre-requisites

Ensure you have the following:
- [ ] A local Git installation and a Go development setup
- [ ] SSH authentication for GitHub (see [here](https://docs.github.com/en/authentication/connecting-to-github-with-ssh))
- [ ] A USB port connection for your hardware key (beware of using a remote connection; the keyholder should not assume that magic occurs during an SSH session)

### Signing pre-work

During a root signing test, keyholders must complete the following steps in order:
- [ ] Fork the [root-signing](https://github.com/sigstore/root-signing) repository by clicking the "fork" button on GitHub. 
- [ ] Test binary build: Set your `${GITHUB_USER}` with your GitHub username and execute the script:
```bash
export GITHUB_USER=${YOUR_GITHUB_USER}
./scripts/step-0.sh
```
This will setup your fork and build the TUF binary to use for metadata generation.
- [ ] (If you are a new keyholder) Test registering your new root key: Do not use an existing key that is already in-use and you need to continue using -- this process will wipe the key! Set the following environment variables, and then follow the steps in [Registering a new root key](../KEYHOLDER.md#registering-a-new-root-key)
```bash
export LOCAL=1
```

**CONFIRM** that you created a new directory under `ceremony/$DATE/keys/` with a new serial numbered `XXXXXX` directory. Run 
```bash
./scripts/verify.sh
```
and confirm that there is some output with `VERIFIED KEY WITH SERIAL NUMBER XXXXXX`.

- [ ] Test signing: Note you will need a test GCP signer. Sigstore keyholders have access to the test KMS key below. You will need to authenticate with GCP. Run the following. 
```bash
export LOCAL=1
gcloud auth application-default login
export TEST_KEYS=gcpkms://projects/project-rekor/locations/global/keyRings/sigstore-root/cryptoKeys
export TIMESTAMP_KEY=$TEST_KEYS/test
export SNAPSHOT_KEY=$TEST_KEYS/test
export REKOR_KEY=$TEST_KEYS/test
export STAGING_KEY=$TEST_KEYS/test
export REVOCATION_KEY=$TEST_KEYS/test
export PREV_REPO=$(pwd)/ceremony/2022-05-10
./scripts/step-1.5.sh
```
Now follow the instructions under [Signing root and targets](../KEYHOLDER.md#signing-root-and-targets).

**CONFIRM** that you created a new directory under `ceremony/$DATE/staged/`. Run 
```bash
export REPO=$(pwd)/ceremony/$(date '+%Y-%m-%d')
./scripts/verify.sh
```
and make sure that you see 1 valid signature for root and targets.

### Registering a new root key

Pre-requisites:
- [ ] Ensure you have run the following during your current session.
```bash
export GITHUB_USER=${YOUR_GITHUB_USER}
./scripts/step-0.sh
```

You may need to install `libpcslite` to support hardware tokens. See [`go-piv`'s installation instructions for your platform.](https://github.com/go-piv/piv-go#installation).

Run 

```bash
./scripts/step-1.sh
```

This step will reset your hardware key and will set a PIN. Choose a PIN between 6 and 8 characters that you will remember for signing in later steps.

This will output three files (a public key, device certificate, and hardware certificate) in a directory named with your serial number `ceremony/YYYY-MM-DD/keys/${SERIAL_NUM}`.

During the actual ceremony, it will push a PR to the root-signing repository.

Troubleshooting: If you hit the error
```
error: connecting to pscs: the Smart card resource manager is not running
```

then run the following to start the pcsc daemon (note: this may require root access):
```
systemctl start pcscd.service
systemctl enable pcscd.service
```

### Signing root and targets

Pre-requisites:
- [ ] Ensure you have run the following during your current session.
```bash
export GITHUB_USER=${YOUR_GITHUB_USER}
./scripts/step-0.sh
```

After the root and targets metadata is created unsigned with placeholder signature IDs, run

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
