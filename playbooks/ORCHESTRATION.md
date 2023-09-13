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

| Variable      | Description                              | Example                                                                              |
|---------------|------------------------------------------|--------------------------------------------------------------------------------------|
| SNAPSHOT_KEY  | The GCP KMS online key for snapshotting. | `projects/sigstore-root-signing/locations/global/keyRings/root/cryptoKeys/snapshot`  |
| TIMESTAMP_KEY | The GCP KMS online key for timestamping. | `projects/sigstore-root-signing/locations/global/keyRings/root/cryptoKeys/timestamp` |

Ensure that these are the values reflected in the staging snapshot and timestamp [workflow](../.github/workflows/staging-snapshot-timestamp.yml).

**NOTE** Ensure that the automated [snapshot and timestamp job](../.github/workflows/stable-snapshot-timestamp.yml) has triggered within the last 5 days. During the week of the ceremony, the job will not push automated updates, and expects the ceremony to be completed within one week. This is to ensure that snapshotting the main ceremony branch does not result in a merge conflict with the new ceremony event. See [693](https://github.com/sigstore/root-signing/issues/693).

## Step 1: Root Key Updates (Optional)

Like mentioned in [Key configuration](#key-configuration), each root key corresponds to a subfolder named by its serial number. The [initialization](#step-2-initializing-a-root-and-targets) script automatically picks up any new subfolders and adds them to the root keys. Any subfolders that are removed are revoked from the root.

### Adding a Root Key

Instruct any new root keyholder to follow [Registering a new root key](keyholders/NEW_SIGNER.md#registration).

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

This step initializes or stages a new root and targets file according
to the pre-work and configuration. Note that _if_ a target should be
updated or a new target to be added, those changes has to be in the
`main` branch before the ceremony starts. The staging of a new root
always happens from the `main` branch, **not** from the ceremony
branch. The GitHub workflow performing this step is
[initialize.yml](../.github/workflows/initialize.yml). Invoke this
workflow with the following parameters:

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

## Step 3: Add delegation (Optional)

This step will add a delegated role to the top-level targets that is
controlled by an external GitHub repository. Coordinate with the
delegation keyholder to run the `add-delegation` command (see
https://github.com/sigstore/root-signing/blob/main/cmd/tuf/app/add-delegation.go).
When creating the delegation with the command, a `target-meta`
file has to be provided that lists the targets, similar to adding
the top level targets, see
[here](ORCHESTRATION.md#targets-and-delegation-configuration) for an
example.
After the delegation metadata is added and signed, the delegation
keyholder should open a PR against the ceremony branch.
The name of the PR MUST be `feat/add-delegation for
<delegation-name>`.

As part of running the `add-delegaton` command, a POP (proof of
possession) has to be generated too. The computed POP should be stored
in `${REPO}/staged/${FORK_POINT}.sig`, where the fork point is the
fork point from `main` and the ceremony branch. This fork point is
also used as the nonce when computing the POP (via `tuf key-pop-sign`,
see below for an example).

The delegation keyholder would run these commands (on a branch based
on the ceremony branch):

Create the delegation metadata
```shell
$ ./tuf add-delegation -name ${DELEGATION_NAME} \
      -public-key ${PUB_KEY_REF} \
      -target-meta delegate-meta.yaml \
      -repository repository
```

```shell
$ ./tuf sign \
      -roles ${DELEGATION_NAME} \
      -key ${KEY_REF} \
      -repository ${REPO}
```

```shell
$ FORK_POINT=$(git merge-base origin/main "${BRANCH}") \
      ./tuf key-pop-sign \
      -key ${KEY_REF} \
      -challenge ${DELEGATION_NAME} \
      -nonce ${FORK_POINT} > ${REPO}/staged/${FORK_POINT}.sig
```
Here `BRANCH` is the ceremony branch, not the branch for the delegation.

When the PR is created, it will trigger the POP verify
[workflow](../.github/workflows/delegation-pop-verify.yml).

To manually verify the POP, run `./scripts/step-0.sh && ./scripts/dpop-verify.sh ${PR_NUM}
${DELEGATION_NAME}`. Don't forget to ensure that `./scripts/verify.sh
$PR` runs on the PR to validate the `targets.json` has a valid
format and that the delegation metadata is properly signed.
Assuming verification passes, merge the PR against the ceremony branch.

## Step 4: Update delegation(s) (Optional)

After the new (updated) repository is staged, the delegations are
_not_ carried over, they must be added back manually. This operation
should be coordinate with a key owner of the delegatee.

The recommended flow is that a key owner of the delegatee performs the
follwoing steps in another branch then the ceremony branch, and
prepares a PR, this [this
PR](https://github.com/sigstore/root-signing/pull/955) for an
example.

:information_source: Identify the key id for the public key for the
delegation,
e.g. `a89d235ee2f298d757438c7473b11b0b7b42ff1a45f1dfaac4c014183d6f8c45`.

Use this key id to extract the public key for the delegate, as it's
needed when the delegation is added back:

```shell
$ cat repository/repository/targets.json | \
      jq -r \
      '.signed.delegations.keys["a89d235ee2f298d757438c7473b11b0b7b42ff1a45f1dfaac4c014183d6f8c45"].keyval.public' \
      > delegate.pem
```

:warning: Updating a delegation **must not** change the key, only the
metadata, hence it's important to use the known key.

Now add the previous delegation file and update it:

```shell
$ cp repository/repository/registry.npmjs.org.json repository/staged
$ cp -r /path/to/delegated/targets .
$ cp /path/to/delegate-meta.yaml .
$ tuf add-delegation \
      -repository repository \
      -name ${DELEGATION_NAME} \
      -target-meta delegate-meta.yaml \
      -public-key delegate.pem
```

This stages the updated version of the delegation, but the version is
**not** yet incremented, and it's not signed.

Sign the delegation and increment the version:

```shell
$ tuf sign \
      -roles ${DELEGATION_NAME} \
      -key ${KEY_REF}
      -repository ./repository \
      -bump-version
```

Create a new branch, add and commit the updated metadata files and
targets, then create a PR:

```
$ git checkout -b update-delegate-${DELEGATION_NAME}
$ git add repository/staged/targets.json
$ git add repository/staged/${DELEGATION_NAME}.json
$ git add repository/staged/targets/${DELEGATION_NAME}
$ git commit --signoff -m "Updated delegation ${DELEGATION_NAME}"
$ git push origin update-delegate-${DELEGATION_NAME}
```

Now a PR can be created agianst the ceremony branch. Review and
approve if all looks good.

When reviewing this PR, it's important for reviewers to verify the
following (against the currently published TUF root):
* Public key has not changed
* Delegation name is the same
* Version has been incremented
* Expiration date is the expected

As part of the review, the signature should be verified too, like
this:
```shell
$ make verify
$ ./verify repository --repository ./repository --staged
...
Verifying <DELEGATION NAME>...
	Success! Signatures valid and threshold achieved
	<DELEGATION NAME> version 9, expires 2024/03/12
...
```

## Step 5: Hardware Key Signing

Next, the root and targets file must be signed. Ask each root
keyholder to follow [Signing root and
targets](keyholders/EXISTING_SIGNER.md#signing).

This will modify `root.json` and `targets.json` with an added signature.

Verify their PRs with the PR number as the argument to the script:

```bash
./scripts/verify.sh $PR
```

You should expect 1 signature added to root and targets on each PR.

After each of the root keyholder PRs are merged, run verification at the head of the ceremony branch:

```bash
./scripts/verify.sh
```

and verify that the root and targets are fully signed.

## Step 6: Snapshotting and Timestamping

Next, the metadata will need to be snapshotted and timestamped. Invoke the staging snapshot and timestamp GitHub workflow [staging-snapshot-timestamp.yml](../.github/workflows/staging-snapshot-timestamp.yml) with the following parameters:

* `branch`: The branch you created for the ceremony, like `ceremony/YYYY-MM-DD`.
* `repo`: The repository folder to trigger this action against, likely the default `repository/` suffices.

This will create a PR signing the snapshot and timestamp files and committed the files. You should see changes in the files in `repository/repository/**` on the branch.

Verify the expirations and the signatures:

```bash
./scripts/verify.sh $PR
```

Note: You cannot test this step locally against the current staged repository, since the snapshot and timestamp keys are only given permissions to the GitHub Workflows. However, under the hood, the workflow is running `./scripts/step-3.sh` and `./scripts/step-4.sh`. If you initialize a ceremony with local testing keys, this action will work.

## Step 7: Publication

Once the PR from [Step 3](#step-3-snapshotting-and-timestamping) is merged, a [workflow](../.github/workflows/sync-ceremony-to-main.yml) will automatically create a PR merging the changes on the completed ceremony branch to main.

Submitting this PR will trigger a push to the preproduction GCS bucket, so ensure that this PR is verified and ready to be pushed!

## Post-ceremony Steps

1. The preproduction GCS bucket will need to be manually synced to the GCS production bucket as of [916](https://github.com/sigstore/root-signing/pull/916).

2. If any root keyholders have changed, update the [current root keyholders](https://github.com/sigstore/root-signing#current-sigstore-root-keyholders) with their name, key ID, and location of their key material.

3. If any targets have changed, update them and their usage in the table containing the [repository structure](https://github.com/sigstore/root-signing#tuf-repository-structure).

4. Announce the root rotation on twitter and the community meeting, and thank the keyholders!

5. Schedule the next root signing event one month before expiration on the calendar. Check [here](https://github.com/sigstore/root-signing/blob/e3f1fe5e487984f525afc81ac77fa5ce39737d0f/cmd/tuf/app/init.go#L29) for root expiration. Schedule a testing event for the week before.

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
