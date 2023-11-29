# Managing timestamp/snapshot metadata

## Background

Timestamp metadata has a short expiration (1 week), so we must regenerate the metadata frequently. Snapshots are also short-lived (3 weeks).

There is a weekly [GitHub Actions cron job](https://github.com/sigstore/root-signing/blob/main/.github/workflows/stable-snapshot-timestamp.yml)
that regenerates the snapshot and timestamp metadata. The GHA will create a PR ([example](https://github.com/sigstore/root-signing/pull/543))
with the metadata, using a Cloud KMS key to sign the metadata. There is also a weekly
[GitHub Actions cron job](https://github.com/sigstore/root-signing/blob/main/.github/workflows/stable-timestamp.yml) to regenerate
timestamp metadata midweek.

After being approved and merged, the metadata is [synced](../.github/workflows/sync-main-to-preprod-and-prod.yml)
to the preproduction and prod GCS buckets. Probers run against the buckets to:

* Verify an artifact using the production TUF repository [here](https://github.com/sigstore/sigstore-probers/blob/main/.github/workflows/reusable-prober.yml#L245-L304)
* Verify the repository and the expiration of the metadata [here](https://github.com/sigstore/sigstore-probers/blob/main/.github/workflows/reusable-prober.yml#L106-L170)

Note the staging environment refers to the `sigstage.dev` environment, not the preproduction environment.

After one day, another GHA runs to [sync](../.github/workflows/sync-preprod-to-prod.yml)
the preproduction bucket to the production bucket.

## Manually creating new snapshot/timestamp metadata

If the snapshot/timestamp GHA needs to be manually run, you can do so by:

* Navigating to the [snapshot/timestamp workflow](https://github.com/sigstore/root-signing/actions/workflows/stable-snapshot-timestamp.yml)
* Select "Run workflow" and click "Run workflow"

You can also run the timestamp-only workflow manually:

* Navigating to the [timestamp workflow](https://github.com/sigstore/root-signing/actions/workflows/stable-timestamp.yml)
* Select "Run workflow" and click "Run workflow"

After the PR is created, approve and merge.

Note: You will need maintainer permissions to run the workflow.

## Syncing preproduction and production

After manually creating new metadata, if the timestamp is nearing expiration (<= 3 days), then you will need to manually sync preproduction and production.
Otherwise, the timestamp in the production bucket will expire before preproduction is synced.

After merging the PR, check that the [sync of the main branch to the preprod and prod buckets](https://github.com/sigstore/root-signing/actions/workflows/sync-main-to-preprod-and-prod.yml) has finished.
Wait until the preproduction probers are healthy.

After that is done, manually run the [sync preprod to prod](https://github.com/sigstore/root-signing/actions/workflows/sync-preprod-to-prod.yml)
GHA:

* Select "Run workflow"
* Click "Run workflow" 

Note: You will need maintainer permissions to run the workflow.
