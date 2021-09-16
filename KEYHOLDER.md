# TUF Generation

0. **The keyholders and the conductor** should fork [this](https://github.com/sigstore/root-signing) git repository by clicking the "fork" button on GitHub. Then, set your `${GITHUB_USER}` with your GitHub username and execute the script:

```
export GITHUB_USER=${YOUR_GITHUB_USER}
./scripts/step-0.sh
```

The conductor should also set the online snapshot and timestamp keys, and the previous repository if it exists. The key references should have the a go-cloud style URI like `gcpkms://<some key>`.

```
export PREV_REPO=${PREVIOUS CEREMONY REPO}
export SNAPSHOT_KEY=${SNAPSHOT_KEY_REFERENCE}
export TIMESTAMP_KEY=${TIMESTAM_KEY_REFERENCE}
```

This will setup your fork and build the TUF binary to use for metadata generation.

You may need to install `libpcslite` to support hardware tokens. See [`go-piv`'s installation instructions for your platform.](https://github.com/go-piv/piv-go#installation).


1. **Each keyholder** should insert their hardware token and run

```
./scripts/step-1.sh
```

You will be prompted to reset your hardware key and set a PIN. Choose a PIN between 6 and 8 characters that you will remember for signing in later steps.

This will output three files (a public key, device certificate, and hardware certificate) in a directory named with your serial number `ceremony/YYYY-MM-DD/keys/${SERIAL_NUM}`.

**Keyholders** should remove their hardware token.


1.5. After all keys are merged, **the conductor** should initialize the TUF repository and add the targets. From this directory:
```
./scripts/step-1.5.sh
TUF repository initialized at  $REPO
Created target file at  $REPO/staged/targets/$TARGET
```

You should see the following directory structure created in `ceremony/YYYY-MM-DD/staged/`.
```
$REPO
├── keys
├── repository
└── staged
    ├── root.json
    ├── targets
    │   └── $TARGET
    ├── targets.json
```

Each metadata file will be populated with a 6 month expiration and placeholder empty signatures corresponding to the KEY_IDs generated in step 1. The `root.json` will specify all 5 keys for each top-level role with a threshold of 3. 

2. Signing root and targets: each **keyholder** should insert their hardware token and sign the root and targets file by running:
```
./scripts/step-2.sh
```

This will prompt your for your PIN twice to sign `root.json` and `targets.json`. This will populate a signature for your key id in the `signatures` section for these two top-level roles.

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

**Keyholders** should remove their hardware token.

3. The **conductor** should initiate the scripts to sign snapshot and timestamp with the online keys after all PRs from step 2 are merged:
```
./scripts/step-4.sh
```

4. After all PRs are merged, the **conductor** can verify and publish the metadata!

```
$ ./scripts/step-4.sh
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
```
