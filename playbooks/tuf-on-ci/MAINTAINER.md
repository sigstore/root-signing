# Maintainer manual for Sigstore root-signing

## Setup

Maintainers need push access to the upstream repository (as signing events are started by
pushing `sign/*` branches). Follow the signer setup instructions except in your
`.tuf-on-ci-sign.ini` set push-remote to the upstream repo as well:
  ```
  [settings]
  user-name = @<YOUR-GITHUB-USER-NAME>
  push-remote = upstream
  pull-remote = upstream
  ```

## Metadata maintenance

Metadata can be modified with two ways: Modifying delegations by running
`tuf-on-ci-delegate` or by modifying the artifacts.
* Comments should always be added in the signing event PR to keep signers aware of changes
* Multiple changes can be done within a single signing event: These changes can modify different
  roles and artifacts

### Modifying delegations (signers, thresholds, etc)

Signers, thresholds, expiry periods and online keys can be modified with the
`tuf-on-ci-delegate <signing-event> <role>` command.
* This will create a commit with the changes, pushes this to a branch on the upstream repo
* GitHub will suggest to "Create a pull request by visiting URL" but this is not required:
  a signing event PR is automatically opened by the signing-event workflow
* The signing event argument must start with "sign/" but can be otherwise freely chosen:
  It will be used as branch name.
* The signing-event workflow will add a comment naming the signers who need to act:
  Remember to document your changes in a signing event PR comment.

#### Examples

<details>
  <summary>Remove a root signer and add another</summary>
    Remove @jku and and add @a-new-signer as signer. The resulting signing event
    will first request @a-new-signer to accept the invite, and then request all
    signers to sign the change.

    ```
    $ tuf-on-ci-delegate sign/add-a-signer root
    Remote branch not found: branching off from main
    Signing event sign/add-a-signer (commit 0b0461f)
    Modifying delegation for root

    Configuring role root
    1. Configure signers: [@jku, @kommendorkapten, @joshuagl, @mnm678], requiring 2 signatures
    2. Configure expiry: Role expires in 91 days, re-signing starts 35 days before expiry
    Please choose an option or press enter to continue: 1
    Please enter list of root signers [@jku, @kommendorkapten, @joshuagl, @mnm678]: @a-new-signer, @kommendorkapten, @joshuagl, mnm678
    Please enter root threshold [2]: 
    1. Configure signers: [@a-new-signer, @kommendorkapten, @joshuagl, @mnm678], requiring 2 signatures
    2. Configure expiry: Role expires in 91 days, re-signing starts 35 days before expiry
    Please choose an option or press enter to continue: 
    Confirm user presence for key ECDSA-SK SHA256:Ca1J+gvZjwnq4UGRyuRzwdJj9tpYtAiweSLtcRui5nA
    User presence confirmed
    Enumerating objects: 10, done.
    Counting objects: 100% (10/10), done.
    Delta compression using up to 8 threads
    Compressing objects: 100% (6/6), done.
    Writing objects: 100% (6/6), 725 bytes | 725.00 KiB/s, done.
    Total 6 (delta 2), reused 0 (delta 0), pack-reused 0 (from 0)
    remote: Resolving deltas: 100% (2/2), completed with 2 local objects.
    remote: 
    remote: Create a pull request for 'sign/add-a-signer' on GitHub by visiting:
    remote:      https://github.com/jku/tuf-on-ci-sigstore-test/pull/new/sign/add-a-signer
    remote: 
    To ssh://github.com/jku/tuf-on-ci-sigstore-test.git
    * [new branch]      HEAD -> sign/add-a-signer

    ```
</details>

### Modifying artifacts (e.g. trusted_root.json)

Artifact modifications can be done with plain git:
* make a commit that modifies a file in `targets/`, push this change to a signing
  event branch. Branch name must start with "sign/" but can be otherwise freely chosen
* GitHub will suggest to "Create a pull request by visiting URL" but this is not required:
  a signing event PR is automatically opened by the signing-event workflow. This PR
  will include the required metadata changes
* The signing-event workflow will add a comment naming the signers who need to act:
  Remember to document your changes in a signing event PR comment

If the legacy custom metadata needs to be modified, there is another manual step:
* Once the signing-event workflow has made the targets metadata changes, you can pull the
  branch, modify the custom metadata manually, and push a new commit into the branch
* Again, keep the signers informed about the changes with a PR comment
