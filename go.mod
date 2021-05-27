module github.com/asraa/sigstore-root

go 1.16

require (
	github.com/peterbourgon/ff/v3 v3.0.0
	github.com/sigstore/cosign v0.4.1-0.20210511181543-454839975345
	github.com/sigstore/sigstore v0.0.0-20210427115853-11e6eaab7cdc
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613
	github.com/theupdateframework/go-tuf v0.0.0-20201230183259-aee6270feb55
)

replace github.com/sigstore/cosign => github.com/sigstore/cosign v0.4.1-0.20210520222622-06657c5c2de9
