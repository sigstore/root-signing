This directory contains the programs needed to generate and verify SigStore root keys and create signed TUF metadata. 

### Ceremony Overview
At the end of the ceremony, new repository metadata will be written to a `ceremony/YYYY-MM-DD/repository` directory.

The ceremony will be completed in four rounds:

![image](https://user-images.githubusercontent.com/5194569/122459506-ffd65e80-cf7e-11eb-8915-e10ac6b50594.png)

* Round 1: Add Key
* Round 2: Sign Root & Targets
* Round 3: Sign Snapshot
* Round 4: Sign Timestamp

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

## Current Sigstore Root Keyholders 
* Bob Callaway [June 2021 - present]
* Dan Lorenc [June 2021 - present]
* Luke Hinds [June 2021 - present]
* Marina Moore [June 2021 - present]
* Santiago Torres-Arias [June 2021 - present]

## Emeritus Sigstore Root Keyholders
* None yet!




