This directory contains the programs needed to generate and verify Sigstore root keys and create signed TUF metadata. 

## TUF Repository Structure

The current published repository metadata lives in the [repository](/repository/repository) subfolder of this GitHub repository. In this repository, you will find the top-level TUF metadata files, delegations, and target files. 

* [root.json](repository/repository/root.json): This is the current `root.json`. It is signed by at least 3 out of the 5 [current root keyholders](https://github.com/sigstore/root-signing#current-sigstore-root-keyholders). Other signing keys endorsed by this root include:
  * A set of 5 targets keyholders (the root keyholders).
  * An online [snapshotting key](https://github.com/sigstore/root-signing/blob/57ac5cd83b90ff97af78db846eea2525eb0eee81/repository/repository/root.json#L87-L97) located at `projects/project-rekor/locations/global/keyRings/sigstore-root/cryptoKeys/snapshot`. You can verify the hex-encoded public key value with the following code:
  ```
  $ gcloud kms keys versions get-public-key 1 --key snapshot --keyring sigstore-root --location global --project project-rekor | openssl ec -pubin -noout -text 
  ```
  * An online [timestamping key](https://github.com/sigstore/root-signing/blob/57ac5cd83b90ff97af78db846eea2525eb0eee81/repository/repository/root.json#L32-L42) located at `projects/project-rekor/locations/global/keyRings/sigstore-root/cryptoKeys/timestamp`
   ```
  $ gcloud kms keys versions get-public-key 1 --key timestamp --keyring sigstore-root --location global --project project-rekor | openssl ec -pubin -noout -text 
  ```
* [targets.json](repository/repository/targets.json): This is the list of trusted `targets.json` endorsed by the 5 root keyholders. It includes:
  * [fulcio_v1.crt.pem](repository/repository/targets/artifact.pub): This is the [Fulcio](https://github.com/sigstore/fulcio) root certificate used to issue short-lived code signing certs. It is hosted at `https://fulcio.sigstore.dev`. You can `curl` the running root CA to ensure it matches the TUF root using `curl -v https://fulcio.sigstore.dev/api/v1/rootCert`
  * [fulcio.crt.pem](repository/repository/targets/artifact.pub): This is the 
  * [rekor.pub](repository/repository/targets/artifact.pub): This is the [Rekor](https://github.com/sigstore/rekor) public key used to sign entries and the tree head of the transparency log. You can retrieve the public key to ensure it matches with `curl -H 'Content-Type: application/x-pem-file' https://rekor.sigstore.dev/api/v1/log/publicKey`.
  * [rekor.0.pub](repository/repository/targets/artifact.pub): This is a dupe of `rekor.pub` and will be removed in the next root-signing event.
  * [ctfe.pub](repository/repository/targets/artifact.pub): Certificate Transparency log key that is used for certificates issued by Fulcio.
  * [artifact.pub](repository/repository/targets/artifact.pub): Key that signs Cosign releases.
* [snapshot.json]((repository/repository/snapshot.json)): This snapshot the valid metadata files. It has a lifetime of 2 weeks and is resigned by a [GitHub workflow](https://github.com/sigstore/root-signing/blob/main/.github/workflows/snapshot-timestamp.yml).
* [timestamp.json]((repository/repository/timestamp.json)): The timestamp refreshes the most accurate snapshot file. It has a lifetime of 2 weeks and is resigned by a [GitHub workflow](https://github.com/sigstore/root-signing/blob/main/.github/workflows/snapshot-timestamp.yml).


### Root locations

The current root is published on a GCS bucket located at `https://storage.googleapis.com/sigstore-tuf-root`.


## Current Sigstore Root Keyholders 
* Bob Callaway [June 2021 - present]:
  - TUF Key ID `f505595165a177a41750a8e864ed1719b1edfccd5a426fd2c0ffda33ce7ff209`: Yubikey material located [here](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/15938791).
* Dan Lorenc [June 2021 - present]
  - TUF Key ID `2f64fb5eac0cf94dd39bb45308b98920055e9a0d8e012a7220787834c60aef97`: Yubikey material located [here](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/13078778).
* Luke Hinds [June 2021 - present]
  - TUF Key ID `bdde902f5ec668179ff5ca0dabf7657109287d690bf97e230c21d65f99155c62`: Yubikey material located [here](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/14454335).
* Marina Moore [June 2021 - present]
  - TUF Key ID `eaf22372f417dd618a46f6c627dbc276e9fd30a004fc94f9be946e73f8bd090b`: Yubikey material located [here](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/14470876).
* Santiago Torres-Arias [June 2021 - present]
  - TUF Key ID `f40f32044071a9365505da3d1e3be6561f6f22d0e60cf51df783999f6c3429cb`: Yubikey material located [here](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/15938765).

### Ceremony Overview
At the end of the ceremony, new repository metadata will be written to a `ceremony/YYYY-MM-DD/repository` directory.

The ceremony will be completed in five rounds:

![image](https://user-images.githubusercontent.com/5194569/122459506-ffd65e80-cf7e-11eb-8915-e10ac6b50594.png)

* Round 1: Add Key
* Round 2: Sign Root & Targets
* Round 3: Sign Delegations
* Round 4: Sign Snapshot & Timestamp

There will be an interim step 1.5 to initialize the TUF metadata and a final step 5 to publish it.


### Ceremony Instructions
Before starting the root key ceremony, the community should:
* Designate the 5 root **keyholders**
* Elect one participant (not necessarily a keyholder) as the **conductor**
* Identify the targets to sign and update the `targets/` directory (these may include Fulcio's CA certificate, the rekor transparency log key, the CTFE key, and SigStore's artifact signing key)
* Identify the online keys for snapshot and timestamp roles. The key references should be updated in `scripts/step-1.5.sh`.

If you are a keyholder or ceremony conductor, follow instructions [KEYHOLDER.md](KEYHOLDER.md).

If you are a verifier, follow instructions at [VERIFIER.md](VERIFIER.md).

### Acknowledgements
Special thanks to Dan Lorenc, Trishank Kuppusamy, Marina Moore, Santiago Torres-Arias, and the whole SigStore community! 

## Emeritus Sigstore Root Keyholders
* None yet!




