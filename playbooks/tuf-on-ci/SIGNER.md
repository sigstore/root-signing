# Keyholder manual for Sigstore root-signing

## Keyholder requirements

1. Availability: keyholders are expected to be available for scheduled signing events
(2-3 times a year) as well as unexpected signing events: if you go on longer travel,
please make sure you either have your signing hardware with you or notify the
sigstore-keyholders Slack channel beforehand.

2. Obtaining a Yubikey for use as a signing key. If you need support to obtain one, please reach
out to one of the maintainers or through Slack. Key configuration is described below.

3. Participation in signing events! Signing events will be announced in sigstore-keyholders
Slack channel and in the signing event PRs. Typically keyholders are expected to review the signing
event changes, sign and create a PR for their signature once per signing event.


## One-time setup for new keyholders

### Yubikey configuration

Generate a PIV Digital Signature key on your hardware key if you don't have one yet.

Using [Yubikey Manager](https://www.yubico.com/support/download/yubikey-manager/)
this is possible in _Applications -> PIV -> Configure certificates -> Digital signature_.
Another option is `cosign piv-tool`.

### Software install

* Install tuf-on-ci-sign
  ```
  # this example uses a virtualenv: feel free to install tuf-on-ci-sign elsewhere
  python3 -m venv ~/.venvs/tuf-on-ci-sign
  source ~/.venvs/tuf-on-ci-sign/bin/activate
  pip install tuf-on-ci-sign

  # If you are on MacOS and the install fails, you can try
  #   brew install swig
  ```
* Install Yubicos PKCS#11 module
  * on Debian `sudo apt install ykcs11`
  * on MacOS `brew install yubico-piv-tool`

### Repository setup

* Fork the repository on github: https://github.com/sigstore/root-signing/fork
* clone your fork and add the upstream as a remote:
  ```
  git clone https://github.com/<YOUR-GITHUB-USER-NAME>/root-signing.git
  cd root-signing
  git remote add upstream https://github.com/sigstore/root-signing.git
  ```
* Create `.tuf-on-ci-sign.ini` with this content:
  ```
  [settings]
  user-name = @<YOUR-GITHUB-USER-NAME>
  push-remote = origin
  pull-remote = upstream
  ```
  If you used an already existing checkout, please make sure the remote names make sense: `push-remote`
  should be your fork, `pull-remote` should be the upstream root-signing repository.

### Smoke test

In your root-signing directory, try to sign in a non-existent signing event:
```bash
$ tuf-on-ci-sign sign/just-testing
Remote branch not found: branching off from main
Signing event sign/just-testing (commit xxxxxxx)
Nothing to do.
$
```

This verifies that `tuf-on-ci-sign` should be ready for signing.

## Signing

When a signing event asks you to sign (or to accept an invite):
* Read the signing event PR comments to find out the purpose and content of this signing event
* If the artifacts in `targets/` (such as `targets/trusted_root.json`) are modified, verify
  that the proposed changes are sensible
* Change into `root-signing` directory
* Enter your virtualenv if you use one: `source ~/.venvs/tuf-on-ci-sign/bin/activate`
* Run signing tool: `tuf-on-ci-sign <SIGNING-EVENT>`
  * if you are accepting an invite, choose "Yubikey" as your key type
  * if you are signing, review the changes (if needed, use GitHub UI or another terminal to see the signing event branch content)
  * Signing automatically commits the signature and pushes it to a branch on your fork
* After signing, click the provided link to create a PR to the signing event branch
