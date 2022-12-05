# Managing timestamp/snapshot metadata

## Background

Timestamp metadata has a short expiration (2 weeks), so we must regenerate the metadata frequently. Snapshots are also short-lived (3 weeks).

There is a weekly [GitHub Actions cron job](https://github.com/sigstore/root-signing/blob/main/.github/workflows/stable-snapshot-timestamp.yml)
that regenerates the snapshot and timestamp metadata. The GHA will create a PR ([example](https://github.com/sigstore/root-signing/pull/543))
with the metadata, using a Cloud KMS key to sign the metadata.

After being approved and merged, the metadata is [synced](https://github.com/sigstore/root-signing/blob/main/.github/workflows/sync.yml)
to a preproduction GCS bucket. Probers run against the bucket to (Repo only visible to Sigstore infrastructure developers):

* Verify an artifact using the production TUF repository [here](https://github.com/sigstore/public-good-instance/blob/main/.github/workflows/reusable-prober.yml#L220-L249)
* Verify the repository and the expiration of the metadata [here](https://github.com/sigstore/public-good-instance/blob/main/.github/workflows/reusable-prober.yml#L134-L156)

Note the staging environment refers to the `sigstage.dev` environment, not the preproduction environment.

After a few days, another GHA runs to [sync](https://github.com/sigstore/root-signing/blob/main/.github/workflows/sync_to_prod.yml)
the preproduction bucket to the production bucket.

## Manually creating new snapshot/timestamp metadata

If the snapshot/timestamp GHA needs to be manually run, you can do so by:

* Navigating to the [snapshot workflow](https://github.com/sigstore/root-signing/actions/workflows/stable-snapshot-timestamp.yml)
* Select "Run workflow" and click "Run workflow"

After the PR is created, approve and merge.

Note: You will need maintainer permissions to run the workflow.

## Syncing preproduction and production

After manually creating new metadata, if the timestamp is nearing expiration (<= 3 days), then you will need to manually sync preproduction and production.
Otherwise, the timestamp in the production bucket will expire before preproduction is synced.

After merging the PR, check that the [sync](https://github.com/sigstore/root-signing/actions/workflows/sync.yml) to preproduction has finished.
Wait until the preproduction probers are healthy.

After that is done, manually run the [sync preprod to prod](https://github.com/sigstore/root-signing/actions/workflows/sync_to_prod.yml)
GHA:

* Select "Run workflow"
* Check the box for "Whether to manually trigger a sync, otherwise only syncs pre-prod to prod with a 2 day delay"
* Click "Run workflow" 

Note: You will need maintainer permissions to run the workflow.
