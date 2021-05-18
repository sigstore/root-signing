This directory contains scripts needed to verify and generate the key artifacts and the TUF metadata. 
The verification CLI is located in `cmd/verify`.

# TUF Generation

0. Install TUF.
```
pip3 install --user tuf
```

Make sure cosign is on your system path.

1. Each keyholder should provision and add their keys with

```
python3 provision.py
```

Take note of your serial number. Verify that you have added a pubkey, device cert, and key cert in a directory named `<SERIAL_NUMBER>`.
Create a pull request with these files.

Observers can verify the files with the verify CLI in `cmd/verify` and the Yubico root CA 
(located at https://developers.yubico.com/PIV/Introduction/piv-attestation-ca.pem).

```
verify --root piv-attestation-ca.pem --key-directory <PATH_TO_CEREMONY_PRODUCTS>
```

This verifies
* That the hardware key is authentic and came from the manufacturer (using the device cert)
* That the signing key was generated on the device (using the key attestation)
* That the directory where they keys were added match the serial number from the cert (preventing a keyholder from using their key multiple times)

2. When everyone has completed generating their keys, a single keyholder should run the following script to generate and commit unsigned metadata files.

```
python3 generate.py
```

3. Each keyholder should now sequentially sign the metadata files using the script 

```
python3 sign.py <SERIAL_NUMBER>
```

This will generate signatures for each of the metadata files (root, targets, snapshot, timestamp).

TODO: Add verification for the signatures. The final PR should have an additional verification to verify the threshold counts.

4. Publish! Observers can fork to distribute the metadata in multiple locations!

### References

* https://github.com/DataDog/integrations-core/tree/master/datadog_checks_downloader


