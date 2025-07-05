## Sigstore root-signing

root-signing project maintains the TUF repository used to securely deliver the
_Sigstore trust root (trusted_root.json)_ to Sigstore clients.

### Documentation

* [signer manual](playbooks/tuf-on-ci/SIGNER.md) documents the process and requirements from
  keyholder perspective
* [maintainer manual](playbooks/tuf-on-ci/MAINTAINER.md) documents some maintenance aspects
* More technical documentation is included in the [tuf-on-ci](https://github.com/theupdateframework/tuf-on-ci/)
  project that is the software used to run root-signing

### TUF repository status

The repository is published for sigstore client consumption at https://tuf-repo-cdn.sigstore.dev/.

The metadata sources can be found in [metadata/](metadata/) folder, artifacts (like trusted_root.json) in [targets/](targets/) folder.
See [Operation](#operation) for more details on how to modify these sources.

#### Keyholders

root-signing security relies on keyholders: they should be trusted community members who are willing and able to
perform keyholder duties like verifying new trusted_root.json content and signing in signing events.

All changes to artifacts or metadata require cryptographic signatures from Sigstore keyholders. Current
keyholders, signature requirements and the signing schedule are documented in
[the published repository](https://tuf-repo-cdn.sigstore.dev/)

| Keyholders | Term |
| - | - |
| Lance Ball | July 2025 -  |
| Joshua Lock | July 2022 -  |
| Bob Callaway | June 2021 -  |
| Marina Moore | June 2021 - |
| Santiago Torres-Arias | June 2021 - |

| Emeritus keyholders | Term |
| - | - |
| Luke Hinds | June 2021 - July 2022 |
| Dan Lorenc | June 2021 - July 2025 |

### Operation

The TUF repository is modified in two ways:
1. _signing events_ where keyholders collaborate to sign changes with their personal hardware keys and
2. _online signing_ where the root-signing machinery signs changes using KMS keys

#### Signing events

Signing events are pull requests created and managed by root-signing where keyholders sign proposed changes.
Signing events happen for multiple reasons:
* Maintainer proposes a change to trusted_root.json
* Maintainer proposes a change to repository configuration (current keyholders, signature thresholds, etc)
* root-signing proposes resigning when signatures are close to expiry

In all cases the trigger to a signing event PR being created is a push to a "sign/*" branch (either by
maintainer or a workflow).

There is a separate [root-signing-staging](https://github.com/sigstore/root-signing-staging) repository:
any non-trivial changes (to metadata or the artifacts) should be tested in root-signing-staging before
they are introduced in a root-signing signing event.

#### Online signing

Online signing happens in two situations:
* A signing event PR has been merged
* A online (KMS) signature is close to expiry

In practice online signing happens at least every three days because of online signature expiry.

#### Publishing and automated testing

Online signing leads to a "preproduction" deployment at https://sigstore.github.io/root-signing/.
This is a fully functional TUF repository that is used to run both generic TUF client tests and
Sigstore specific client tests (with cosign and other sigstore clients). Successful tests lead to
production deployment at https://tuf-repo-cdn.sigstore.dev/.

### Workflows

The important workflows in root-signing are:
* `create-signing-events` creates branches for signing events when signatures are close to expiry.
  Runs on schedule
* `signing-event` creates and manages the signing event pull requests. Runs when "sign/*" branches
  are pushed to
* `online-sign` commits and merges online signatures, also dispatches `publish`. Runs on when
  "main" is pushed to (but can be manually dispatched at any time)
* `publish` publishes a test repository to GitHub Pages, runs client tests, and finally publishes
  the repository. Runs on dispatch from `online-sign`

### Acknowledgements
Special thanks to Dan Lorenc, Trishank Kuppusamy, Marina Moore, Santiago Torres-Arias, and the whole SigStore community!

### Initial Root Signing Ceremony
A recording of the signing ceremony is available [here](https://www.youtube.com/watch?v=GEuFsc8Zm9U).
