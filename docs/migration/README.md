# Migration to tuf-on-ci managed TUF repository

The plan is to manage root-signing with [tuf-on-ci](https://github.com/theupdateframework/tuf-on-ci),
just like root-signing-staging. This is documented in https://github.com/sigstore/root-signing/issues/929:
in short it aims to replace the current workflows and code with much smaller workflows that use the
tuf-on-ci actions.

## Summary

Roughly the migration steps are
1. Prepare the GitHub project, workflows, cloud services and the data
   _without disrupting the current operations_
2. Run the initial signing event with tuf-on-ci, do the necessary changes to metadata _without disrupting
   the current operations_: this is achieved by
   * modifying a separate copy of the metadata
   * disabling the metadata publishing from tuf-on-ci
3. If CI tests pass and manual testing looks good: disable current repository workflows, enable metadata
   publishing from tuf-on-ci
4. Remove now obsolete workflows, code and metadata copies

The complication here is that starting step 2 (making a copy of the current metadata) starts a timer as
the online signing machinery will modify metadata every three days. Within that time we need to either:
* Decide the signing event was not successful and restart step 2 at a later date or
* complete steps 2 and 3

The deadline can be extended by a couple of days (as the actual expiry period is ~7 days) but this requires
additional communication with on-call.

## Detailed playbook

### 1. Preparation

All of these steps can be taken without time pressure from the online signing process.

#### Update GitHub project settings

Review root-signing settings in sigstore/community, compare to root-signing-staging settings
and https://github.com/theupdateframework/tuf-on-ci/blob/main/docs/REPOSITORY-MAINTENANCE.md:
Enable things that are required by tuf-on-ci (do not disable the permissions required by legacy root-signing).

Define GitHub variables and secrets:
* Variable GCP_SERVICE_ACCOUNT (this is the online signing service account)
* Variable GCP_WORKLOAD_IDENTITY_PROVIDER (this is the online signing identity provider)
* Secret TUF_ON_CI_TOKEN (this is a sigstore-bot token)

#### Update GCP configuration

The settings should be easy to copy from root-signing-staging, with the exception that there are now two different
service accounts: one for KMS and one for GCS
* KMS: Online signing must be possible from the online-sign workflow from the main branch
* GCS: uploading must be possible from publish workflow from the publish branch

#### Update targets/ directory in git

Currently targets/ does not quite match the repository contents.

#### Add tuf-on-ci workflows to git

Add the workflows but disable scheduled runs and specifically prevent publishing before it is enabled.

#### Add import configuration to git

* Add import script that copies metadata in expected locations and rewrites whitespace (so future diffs are manageable)
* Add configuration file for the actual tuf-on-ci import

#### Add/Update playbooks to git

This playbook and manuals for signers and maintainers.

#### Signer orientation

Make sure signers have installed and tested the software, and understand what will be requested during the signing
event -- especially with regards ~3 day timeline to evaluate and make a go/no-go decision.

#### General outreach

Communicate the upcoming changes to Sigstore community

### 2. Initial signing event

The initial signing event can be restarted if it looks like there are issues that cannot be solved within the ~3 day timeline.
It makes sense to start the signing event just after online signing to make that window as large as possible.

#### Start signing event

* Maintainer: Run the import script to copy current metadata to new directory, merge to main
* Maintainer: run tuf-on-ci-import-repo with the prepared configuration file: this starts a signing event
* Maintainer: run tuf-on-ci-delegate to modify online roles in the signing event:
  * they should use the same key (this makes sense in root-signing as both snapshot and timestamp are
    signed in same process with same access credentials)
  * signing period is 4 days, expiry 7 days

#### Signing event

Every signer in the repository needs to sign the signing event at this point by running `tuf-on-ci-sign`:
this is required because the keyids of all keys have changed (the actual key content has not).

** TODO: This includes the npmjs signer: it is unclear how this signing happens at this point**

In the same signing event PR all workflows can be enabled. Note that:
* deploy-to-gcs step should be still left commented in publish workflow
* the legacy workflows can keep running (although online-sign can be disabled if schedule is tight )

#### First online sign

Merging the signing event PR
* triggers online signing
* publishes the test "preprod" repository sigstore.github.io/root-signing/
* runs all client tests against that repository
* does NOT publish to production GCS yet as this is still disabled

Manual client tests against this repository are now possible.


### 3. Migration

#### Disable publishing from legacy workflows

Make sure that legacy publishing can no longer happen under any circumstances

#### Publish to GCS

* Enable GCS Publish by uncommenting lines in publish workflow
* dispatch online-sign workflow manually to kickstart the process
  (this will not actually sign as it's not needed but will trigger publishing)

#### Update sigstore-prober

The reusable prober tuf_preprod_repo input default value should be set to: "https://sigstore.github.io/root-signing"

### 4. Post-migration

#### Remove legacy workflows

#### Update README

* Remove references to obsolete processes
* Update content so it links to the generated repository description

#### Delete code and data that is no longer needed in git

This is almost all directories in the repo (pkg, ceremony, release, config, cmd, scripts, repository) but this can happen piece by piece.

#### Update GitHub project settings

Disable any permissions that were enabled for the legacy workflows but are not needed by tuf-on-ci

#### Remove non-versioned metadata from production GCS bucket

The bucket currently contains URLs like /root.json which are not published by tuf-on-ci and not used by actual TUF clients.

We should remove these files as obsolete (or alternatively add some code that keeps publishing them)

#### Handle preprod bucket GCS

tuf-on-ci uses Github Pages for the same purpose so preprod bucket no longer gets updated.