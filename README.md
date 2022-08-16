This directory contains the programs needed to generate and verify Sigstore root keys and create signed TUF metadata. 

## TUF Repository Structure

The current published repository metadata lives in the [repository](/repository/repository) subfolder of this GitHub repository. In this repository, you will find the top-level TUF metadata files, delegations, and target files. 

* [root.json](repository/repository/root.json): This is the current `root.json`. It is signed by at least 3 out of the 5 [current root keyholders](https://github.com/sigstore/root-signing#current-sigstore-root-keyholders). The top-level signing keys endorsed by the root are:

| Role        | TUF Key ID(s) |  Description | 
| ----- | ------| --------- |  
| Root    | See below | The [offline keyholders](https://github.com/sigstore/root-signing#current-sigstore-root-keyholders).      |
| Targets    | See below | The [offline keyholders](https://github.com/sigstore/root-signing#current-sigstore-root-keyholders), the same as the root keyholders to minimize the number of offline keysets.       |
| Snapshot  | `fc61191ba8a516fe386c7d6c97d918e1d241e1589729add09b122725b8c32451` | A GCP KMS [snapshotting key](https://github.com/sigstore/root-signing/blob/57ac5cd83b90ff97af78db846eea2525eb0eee81/repository/repository/root.json#L87-L97) located at `projects/project-rekor/locations/global/keyRings/sigstore-root/cryptoKeys/snapshot`       |
| Timestamp  | `b6710623a30c010738e64c5209d367df1c0a18cf90e6ab5292fb01680f83453d`| A GCP KMS   [timestamping key](https://github.com/sigstore/root-signing/blob/57ac5cd83b90ff97af78db846eea2525eb0eee81/repository/repository/root.json#L32-L42) located at `projects/project-rekor/locations/global/keyRings/sigstore-root/cryptoKeys/timestamp`      |

* [targets.json](repository/repository/targets.json): This is the list of trusted `targets.json` endorsed by the offline keyholders. It includes:

| Target        |  Description | 
| ----- |--------- |  
| [fulcio_v1.crt.pem](repository/repository/targets/fulcio_v1.crt.pem)   |  This is the [Fulcio](https://github.com/sigstore/fulcio) root certificate used to issue short-lived code signing certs. It is hosted at `https://fulcio.sigstore.dev`. You can `curl` the running root CA chain to ensure the first PEM-encoded certificate matches the TUF root using `curl -v https://fulcio.sigstore.dev/api/v1/rootCert` | 
| [fulcio_intermediate_v1.crt.pem](repository/repository/targets/fulcio__intermediate_v1.crt.pem)   |  This is the [Fulcio](https://github.com/sigstore/fulcio) intermediate certificate used to issue short-lived code signing certs. It is hosted at `https://fulcio.sigstore.dev`. You can `curl` the running CA chain to ensure the second PEM-encoded certificate matches the TUF root using `curl -v https://fulcio.sigstore.dev/api/v1/rootCert` | 
| [fulcio.crt.pem](repository/repository/targets/fulcio.crt.pem)        |  This is the Fulcio root certificate used with an older instance of Fulcio. We maintain this target to verify old certificates but is no longer used to sign newly issued certificates. | 
| [rekor.pub](repository/repository/targets/rekor.pub)        |  This is the [Rekor](https://github.com/sigstore/rekor) public key used to sign entries and the tree head of the transparency log. You can retrieve the public key to ensure it matches with `curl -H 'Content-Type: application/x-pem-file' https://rekor.sigstore.dev/api/v1/log/publicKey`. | 
| [rekor.0.pub](repository/repository/targets/rekor.0.pub)        |  This is a dupe of `rekor.pub` and will be removed in the next root-signing event. | 
| [ctfe.pub](repository/repository/targets/ctfe.pub)        |  Certificate Transparency log key that is used for certificates issued by Fulcio and used to verify signed certificate timestamps (SCTs) for inclusion into the log. | 
| [artifact.pub](repository/repository/targets/artifact.pub) | Key that signs Sigstore project (Cosign, Rekor, Fulcio) releases. |

* [snapshot.json]((repository/repository/snapshot.json)): The snapshot ensures consistency of the metadata files. It has a lifetime of 2 weeks and is re-signed by a [GitHub workflow](https://github.com/sigstore/root-signing/blob/main/.github/workflows/snapshot-timestamp.yml).
* [timestamp.json]((repository/repository/timestamp.json)): The timestamp indicates the freshness of the metadata files. It has a lifetime of 2 weeks and is re-signed by a [GitHub workflow](https://github.com/sigstore/root-signing/blob/main/.github/workflows/snapshot-timestamp.yml).


### Root locations

The current root is published on a GCS bucket located at `https://storage.googleapis.com/sigstore-tuf-root`.

The pre-production root is published on a GCS bucket located at `https://storage.googleapis.com/sigstore-preprod-tuf-root`.


## Sigstore Root Keyholders 

### Current Keyholders

| Keyholder        |  TUF Key ID |  Yubikey Material| Term | 
| ----- |--------- |  --- | ---- |
| Joshua Lock       |  `75e867ab10e121fdef32094af634707f43ddd79c6bab8ad6c5ab9f03f4ea8c90` | [18158855](https://github.com/sigstore/root-signing/ceremony/2022-07-12/keys/18158855)  | July 2022 -  | 
| Bob Callaway        |  `f505595165a177a41750a8e864ed1719b1edfccd5a426fd2c0ffda33ce7ff209` | [15938791](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/15938791)  | June 2021 -  | 
| Dan Lorenc        |  `2f64fb5eac0cf94dd39bb45308b98920055e9a0d8e012a7220787834c60aef97` | [13078778](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/13078778)  | June 2021 -  | 
| Marina Moore        |  `eaf22372f417dd618a46f6c627dbc276e9fd30a004fc94f9be946e73f8bd090b` | [14470876](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/14470876)  | June 2021 -  | 
| Santiago Torres-Arias        |  `f40f32044071a9365505da3d1e3be6561f6f22d0e60cf51df783999f6c3429cb` | [15938765](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/15938765) | June 2021 -  | 

### Emeritus Keyholders
| Keyholder        |  TUF Key ID |  Yubikey Material| Term | 
| ----- |--------- |  --- | ---- |
| Luke Hinds        |  `bdde902f5ec668179ff5ca0dabf7657109287d690bf97e230c21d65f99155c62` | [14454335](https://github.com/sigstore/root-signing/tree/main/ceremony/2021-06-18/keys/14454335)  | June 2021 - July 2022 | 


### Ceremony Overview
At the end of the ceremony, new repository metadata will be written to a `ceremony/YYYY-MM-DD/repository` directory.

The ceremony will be completed in five rounds:

![image](https://user-images.githubusercontent.com/5194569/122459506-ffd65e80-cf7e-11eb-8915-e10ac6b50594.png)

* Round 1: Add Key
* Round 1.5: Initialize TUF metadata 
* Round 2: Sign Root & Targets
* Round 3: Sign Delegations
* Round 4: Sign Snapshot & Timestamp
* Round 5: Publish final repository.

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

## Initial Root Signing Ceremony

A recording of the signing ceremony is available [here](https://www.youtube.com/watch?v=GEuFsc8Zm9U).


