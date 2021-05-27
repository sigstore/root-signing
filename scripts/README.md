This directory contains programs needed to verify and generate the key artifacts and the TUF metadata. 
* The metadata generation go implementation is located in `cmd/metadata`.
* The verification CLI is located in `cmd/verify`.

# TUF Generation

0. **Each keyholder** should:

* Install TUF and build the TUF application
```
$ go get github.com/theupdateframework/go-tuf/cmd/tuf
$ go build -o tuf ./cmd/tuf
```

* Make sure cosign is on your system path.

* Setup their environment with the repository path corresponding to the ceremony's date
```
export REPO=/path/to/this/repository/ceremony/YYYY-MM-DD
```

* Gather the target materials to sign (e.g. rekor tlog key, ctfe tlog key, artifact signing key, and Fulcio CA certificate).

* Designate a participant (not necessarily a keyholder) as the **conductor**.

1. **The conductor** should initialize the TUF repository and add the targets. From this directory:
```
$ ./tuf init -repository $REPO [-targets $TARGET [$TARGET2 $TARGET3 $TARGET4]]
TUF repository initialized at  $REPO
Created target file at  $REPO/staged/targets/$TARGET
```

You should see the following directory structure created
```
$REPO
├── keys
├── repository
└── staged
    ├── root.json
    ├── snapshot.json
    ├── targets
    │   └── $TARGET
    ├── targets.json
    └── timestamp.json
```

**The conductor** should create a PR.

**Keyholders and observers** should verify that the expiration and threshold in each of the the unpopulated metadata files match their expectations (e.g. 4 month expiration and threshold 3).
They should carefully check the target hashes that were added match theirs.

2. **Each keyholder** should pull the PR from step 1 and and provision their keys with
```
$ ./tuf add-key -repository $REPO
[public key info]

Wrote public key data to  $REPO/keys/$SERIAL
```

This will output the path to your key's artifact directory. You will find the pubkey, key certificate, and the device certificate in the folder.

This will also add your key to the unsigned metadata in each target role in `root.json`.

```
   "roles": {
      "root": {
        "keyids": [
          "bdde79c4341bc2e31a6e855dade997a1a2d25c4fd1987c27eb33ed77b14de8af"
        ],
        "threshold": 3
      },
```

Create a pull request with these changes.

**Observers** can verify the files with the verify CLI in `cmd/verify` and the Yubico root CA.

```
$ go build -o verify ./cmd/verify
$ wget https://developers.yubico.com/PIV/Introduction/piv-attestation-ca.pem
$ ./verify --root piv-attestation-ca.pem --key-directory $REPO/keys

2021/05/24 10:42:25 verified key 14833186
```

This verifies
* That the hardware key is authentic and came from the manufacturer (using the device cert)
* That the signing key was generated on the device (using the key attestation)
* That the directory where they keys were added match the serial number from the cert (preventing a keyholder from using their key multiple times)


3. When everyone has completed provisioning their keys, **keyholders** should run three sequentual rounds to sign the metadata files. You will not be able to skip rounds. The script will verify that the previous step's metadata files were signed correctly (with the correct threshold and valid signatures).

a. Round one. Take turns signing root and targets: 

```
$ ./tuf sign -repository $REPO -roles root -roles targets
```

This will add signatures to the root and targets files:
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

b. Round two: 
```
$ ./tuf sign -repository $REPO -roles snapshot
```

c. Round three:
```
$ ./tuf sign -repository $REPO -roles timestamp
```

4. The **conductor** can publish the metadata! **Observers** can fork to distribute the metadata in multiple locations!

```
$ ./tuf publish -repository $REPO
Metadata successfully validated!
```

This will move the finalized metadata to `$REPO/repository`:
```
$REPO
├── keys
│   └── 14833186
│       ├── 14833186_device_cert.pem
│       ├── 14833186_key_cert.pem
│       └── 14833186_pubkey.pem
│   └── [more]
├── repository
│   ├── 0.root.json
│   ├── root.json
│   ├── snapshot.json
│   ├── targets
│   │   └── fulcio.crt.pem
│   ├── targets.json
│   └── timestamp.json
└── staged
    └── targets

```

### References

* https://github.com/DataDog/integrations-core/tree/master/datadog_checks_downloader
