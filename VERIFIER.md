At any point, a verifier can run a script to verify each incoming PR and verify the hashes of the targets files.

0. **Verifiers** should fork [this](https://github.com/sigstore/root-signing) git repository by clicking the "fork" button on GitHub. Then, set your `${GITHUB_USER}` with your GitHub username and set the repository name:

```
export GITHUB_USER=${YOUR_GITHUB_USER}
```

1. To verify a PR, run the script with the pull request ID to verify:

```
./scripts/verify.sh ${PULL_REQUEST_ID}
```

This will download the Yubico root CA. For each key added, it will verify:
* That the hardware key is authentic and came from the manufacturer (using the device cert)
* That the signing key was generated on the device (using the key attestation)
* That the directory where they keys were added match the serial number from the cert (preventing a keyholder from using their key multiple times)

If there is any repository data added in the PR, it will also check signatures in each top-level role.

