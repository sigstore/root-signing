CC @sigstore/tuf-root-signing-codeowners, please have a look.

* The workflow will be automatically retried every six hours: this issue will be updated based on retry results
* If the failure looks like flaky infrastructure, it can be manually re-run. The workflows can also be dispatched manually (recommendation is to avoid dispatching `publish` workflow: dispatch `online-sign` instead and let that workflow dispatch `publish`)
* Failures in `publish` or `online-sign` may be urgent: see expiry dates of current repository in https://tuf-repo-cdn.sigstore.dev/
