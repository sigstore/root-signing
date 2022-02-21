module github.com/sigstore/root-signing

go 1.16

require (
	github.com/peterbourgon/ff/v3 v3.1.2
	github.com/pkg/errors v0.9.1
	github.com/sigstore/cosign v1.5.2
	github.com/sigstore/sigstore v1.1.1-0.20220130134424-bae9b66b8442
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613
	github.com/theupdateframework/go-tuf v0.0.0-20220124194755-2c5d73bebc1c
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

replace github.com/theupdateframework/go-tuf => github.com/asraa/go-tuf v0.0.0-20220210145718-70479943a56c
