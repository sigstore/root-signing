## root-signing infrastructure

### Cloud key management

Online signing for snapshot and timestamp roles uses Google Cloud KMS.
* The current KMS keyid is defined in [root.json](https://github.com/sigstore/root-signing/blob/main/metadata/root.json)
* The used KMS key can be changed with `tuf-on-ci-delegate`: this requires the maintainer to
  have the `cloudkms.publicKeyViewer` role for the new key, and will result in a root signing event.
* The service account and workload identity provider are defined in the
  [online-sign workflow](https://github.com/sigstore/root-signing/blob/main/.github/workflows/online-sign.yml)
* KMS management happens in the public-good-instance repository using the tuf module from
  [sigstore/scaffolding](https://github.com/sigstore/scaffolding)

### GitHub App authentication

Several workflows in this repository require elevated permissions. This is achieved using a GitHub App:
* The App ID is stored in repository variable `TUF_ON_CI_APP_ID` and the private key in repository secret `TUF_ON_CI_APP_PRIVATE_KEY`.
* Required App setup and permissions are documented in tuf-on-ci
  [GitHub App setup documentation](https://github.com/theupdateframework/tuf-on-ci/blob/main/docs/GITHUB-APP-SETUP.md).

### Repository publishing

root-signing publishes a testing repository with GitHub Pages at https://sigstore.github.io/root-signing/.

The final publish is done using Google Cloud Storage and is made available at https://tuf-repo-cdn.sigstore.dev/.
* The service account and workload identity provider are defined in the
  [publish workflow](https://github.com/sigstore/root-signing/blob/main/.github/workflows/publish.yml)
* Other details, like the GCS bucket name, are defined in the
  [deploy-to-gcs workflow](https://github.com/sigstore/root-signing/blob/main/.github/workflows/deploy-to-gcs.yml)
* GCS management happens in the public-good-instance repository using the tuf module from
  [sigstore/scaffolding](https://github.com/sigstore/scaffolding)

