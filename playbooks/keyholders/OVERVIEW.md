# Keyholder Responsibilities

This document outlines the responsibilities of a root keyholder.

## Outline

1. Keyholders MUST subscribe to the [Sigstore Maintainer Calendar](https://calendar.google.com/calendar/u/0?cid=Y19ydjIxcDJuMzJsbmJoYW5uaXFwOXIzNTJtb0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t) for upcoming root signing events. Root signing events are expected to occur about every 4-5 months. The next `v+1` version signing will be scheduled, and the `v+2` version will be tentatively scheduled.

2. A testing event will occur the week before the signing. Keyholders are required to communicate that they have completed the testing ([new](./NEW_SIGNER.md/#testing) and [existing](./EXISTING_SIGNER.md/#testing) testing) to the orchestrator through the [#sigstore-keyholder](https://sigstore.slack.com/archives/C03E4HP6RCK) Slack channel. All testing can occur asynchronously. 

3. Obtaining a Yubikey for use in the root signing key. Yubikeys  If you need support to obtain one, please reach out to one of the maintainers or through Slack. Commands have been tested against the Yubikeys mentioned [here](https://github.com/sigstore/cosign/blob/main/TOKENS.md#tested-devices), although other hardware may be supported.

4. Participation in root signing events! Keyholders are expected to participate in the scheduled root signing events. The steps will be announced in Slack. New keyholders will need to perform two PR creations, and existing keyholders will need to perform a single PR creation. Additionally, keyholders should be "on-call" (available for quick pings during daytime hours) during the root signing window in case there is an issue.
