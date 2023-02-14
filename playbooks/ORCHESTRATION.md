# Orchestration

This playbook describes how to orchestrate a root signing event and a pre-test.

## Pre-work

1. Check the [calendar](https://calendar.google.com/calendar/u/0?cid=Y19ydjIxcDJuMzJsbmJoYW5uaXFwOXIzNTJtb0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t) for upcoming root signing events.

2. Make updates to the targets and delegation configuration files, see [configuration](#targets-and-delegation-configuration). To add a delegation, see [Adding a Delegation](#adding-a-delegation).

3. Double-check the configured [role expirations](https://github.com/sigstore/root-signing/blob/e3f1fe5e487984f525afc81ac77fa5ce39737d0f/cmd/tuf/app/init.go#L28).

4. Create (or ask a root-signing maintainer) to create a new upstream branch named by the ceremony date, for example, `ceremony/YYYY-MM-DD`. For a test ceremony, use `test-ceremony/YYYY-MM-DD`.

## Configuration

Each root signing event occurs inside a branch `ceremony/YYYY-MM-DD` named by the date the signing event started. The branch is only merged into main when the repository is completely signed and validated with any client testing necessary.

The TUF repository is always staged inside the top-level `repository/` folder. This folder contains:

* A `keys/` subfolder containing subdirectories named after Yubikey serial numbers, and containins public key PEMs, key certificates, and device certificates attesting to the hardware key. See [PIV attestation](https://developers.yubico.com/PIV/Introduction/PIV_attestation.html).
* A `repository/` subfolder containing the finalized TUF repository metadata.
* A `staging/` subfolder present during root signing events with staged metadata before publishing.

### Targets and Delegation configuration

For top-level targets and delegation files, a YAML configuration file named `role-metadata.yml` specifies the target files for the role and their [custom metadata](https://theupdateframework.github.io/specification/latest/#custom). We use [Sigstore custom metadata](https://github.com/sigstore/sigstore/blob/ec8f2d403e07a392ea363560d21c31aaee57ba0f/pkg/tuf/client.go#L95) to allow Sigstore clients to identity usage of the targets. Target files are customarily located in the [targets/] subfolder. For example, this defines the target file located at `targets/fulcio_v1.crt.pem` with custom metadata to indicate Fulcio root usage.

```yaml
targets/fulcio_v1.crt.pem:
  sigstore:
    usage: Fulcio
    status: Active
```

### Key configuration

There are two types of keys that are used during root signing.

First, we use hardware Yubikeys for root and target signers. There are 5 [root keyholders](https://github.com/sigstore/root-signing#current-keyholders) each containing one key. A [default threshold](https://github.com/sigstore/root-signing/blob/e3f1fe5e487984f525afc81ac77fa5ce39737d0f/cmd/tuf/app/init.go#L24) of 3 are required to sign the root and targets. This is configurable through a `threshold` flag on [initialization](#step-2-initializing-a-root-and-targets). Configuring the hardware keyholders is done through management of the `keys/` subfolder, see [Key Management](#step-1-root-key-updates-optional) during the ceremony.

Second, online keys on GCP are used for snapshot, timestamp, and delegation roles. These are defined by a [go-cloud](https://gocloud.dev) style URI to refer to the specific provider like `gcpkms://`. See cosign [KMS integrations](https://github.com/sigstore/cosign/blob/main/KMS.md) for details. These are configured through environment variables in the `scripts/` which propagate to command line flags to the binary.

## Step 0: Workflow configuration check

All ceremony orchestration actions use GitHub workflows in this repository. Before beginning the ceremony, ensure that the workflow options are correct.

You will need the following variables for the online signer references described [here](#key-configuration).

| Variable      | Description | Example |
| ----------- | ----------- | ----------- |
| SNAPSHOT_KEY   | The GCP KMS online key for snapshotting.    |     `projects/sigstore-root-signing/locations/global/keyRings/root/cryptoKeys/snapshot`    |
| TIMESTAMP_KEY   | The GCP KMS online key for timestamping.    |  `projects/sigstore-root-signing/locations/global/keyRings/root/cryptoKeys/timestamp`  |

Ensure that these are the values reflected in the staging snapshot and timestamp [workflow](../.github/workflows/staging-snapshot-timestamp.yml).

## Step 1: Root Key Updates (Optional)

Like mentioned in [Key configuration](#key-configuration), each root key corresponds to a subfolder named by its serial number. The [initialization](#step-2-initializing-a-root-and-targets) script automatically picks up any new subfolders and adds them to the root keys. Any subfolders that are removed are revoked from the root.

### Adding a Root Key

Instruct any new root keyholder to follow [Registering a new root key](../KEYHOLDER.md#registering-a-new-root-key)

This will create the following structure.

```bash
${REPO}/keys
└── 89957089
    ├── 89957089_device_cert.pem
    ├── 89957089_key_cert.pem
    └── 89957089_pubkey.pem
```

Verify the PRs with the PR number as the argument to the script:

```bash
./scripts/verify.sh $PR
```

You should expect to see their serial number key verified, which should match the committed subfolder.

### Revoking a Root Key

Removing a root key occurs by removing a key material subfolder. This is done through the [initialization](#step-2-initializing-a-root-and-targets) script by passing the serial number as workflow argument. See below.

## Step 2: Initializing a root and targets

This step initializes or stages a new root and targets file according to the pre-work and configuration. The GitHub workflow performing this step is [initialize.yml](../.github/workflows/initialize.yml). Invoke this workflow with the following parameters:

* `branch`: The branch you created for the ceremony, like `ceremony/YYYY-MM-DD`.
* `revoke_key`: The serial number of a key that should be revoked.
* `repo`: The repository folder to trigger this action against, likely the default `repository/` suffices.
* `draft`: Creates a draft pull request for testing.

Any new key additions from [Step 1](#step-1-root-key-updates-optional) will be picked up.

If you want to test this action locally first, use:

```bash
GITHUB_USER=${GITHUB_USER} ./scripts/step-0.sh
LOCAL=1 ./scripts/step-1.5.sh $revoke_key
```

This copies over old repository metadata and keys from the `${PREV_REPO}`, revokes key `123456`, and then updates a new root and targets according to the configuration. The new PR will create a new `root.json`, `targets.json`, and `targets` subfolder with the desired targets. You should see the following directory structure created:

```bash
$REPO
├── keys
├── repository
└── staged
    ├── root.json
    ├── targets
    │   └── $TARGET
    ├── targets.json
```

Manually check for:

* The expected root and targets expirations.
* The expected root and targets versions.
* The expected root and targets thresholds.
* The expected keyholders and placeholder signatures.
* The expected target files. Check the targets' custom metadata in the targets file.

<!-- TODO: Add playbook for disaster/recovery steps. -->

### Hardware Key Signing

Next, the root and targets file must be signed. Ask each root keyholder to follow [Signing root and targets](../KEYHOLDER.md#signing-root-and-targets).

This will modify `root.json` and `targets.json` with an added signature.

Verify their PRs with the PR number as the argument to the script:

```bash
./scripts/verify.sh $PR
```

You should expect 1 signature added to root and targets on each PR.

After each of the root keyholder PRs are merged, run verification at main:

```bash
./scripts/verify.sh
```

and verify that the root and targets are fully signed.


<!--  
TODO(https://github.com/sigstore/root-signing/issues/398):
Re-instate when delegation work is complete.

## Step 3: Delegations

After root and targets signing, the delegation files must be signed.

```bash
./scripts/step-3.sh
```

This will create a PR signing the delegations. Verify the PR with the PR number as the argument to the script:

```bash
./scripts/verify.sh $PR
```

and check that the delegation was successfully signed.
 -->


## Step 3: Snapshotting and Timestamping

Next, the metadata will need to be snapshotted and timestamped. Run

```bash
./scripts/step-3.sh
```

This will create a PR signing the snapshot and timestamp files. Verify the expirations and the signatures:

```bash
./scripts/verify.sh $PR
```

## Step 4: Publishing

This final step will commit the TUF repository metadata and move it to the top-level folder `repository/repository/`.

```bash
./scripts/step-4.sh
```

This will create a PR moving the files. Verify that the TUF client can update to the new metadata with:

```bash
./scripts/verify.sh $PR
```

## Post-ceremony Steps

1. If any root keyholders have changed, update the [current root keyholders](https://github.com/sigstore/root-signing#current-sigstore-root-keyholders) with their name, key ID, and location of their key material.

2. If any targets have changed, update them and their usage in the table containing the [repository structure](https://github.com/sigstore/root-signing#tuf-repository-structure).

3. Announce the root rotation on twitter and the community meeting, and thank the keyholders!

4. Schedule the next root signing event one month before expiration on the calendar. Check [here](https://github.com/sigstore/root-signing/blob/e3f1fe5e487984f525afc81ac77fa5ce39737d0f/cmd/tuf/app/init.go#L29) for root expiration. Schedule a testing event for the week before.

### Other

#### Encountering a configuration mistake

In case there is a configuration mistake or a breakage that renders a ceremony ineffective, move the half-completed ceremony directory into `ceremony/defunct`. This may help avoid keyholders from pointing to an invalid ceremony directory when signing.

#### Adding a Delegation

1. Add an environment variable for the delegation key named `$DELEGATION_KEY` in [./scripts/step-1.5.sh].

2. Create a `./config/$DELEGATION-metadata`.yml file, see [Target and Delegation configuration](#targets-and-delegation-configuration). 

3. Edit [./scripts/step-1.5.sh] to add the delegation after the root and targets are setup via `tuf init`, with a command like:

```bash
# Add $DELEGATION delegation
./tuf add-delegation -repository $REPO -name "$DELEGATION" -key $DELEGATION_KEY -target-meta config/$DELEGATION-metadata.yml -path $PATH
```

The optional `-path $PATH` specifies any [paths](https://theupdateframework.github.io/specification/latest/#delegation-role-paths) that describes paths the delegated role is trusted to provide.

4. Update the [../README.md] with the delegation information in [Repository Structure](../README.md#tuf-repository-structure).