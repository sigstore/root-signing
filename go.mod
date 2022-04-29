module github.com/sigstore/root-signing

go 1.16

require (
	github.com/peterbourgon/ff/v3 v3.1.2
	github.com/pkg/errors v0.9.1
	github.com/sigstore/cosign v1.8.0
	github.com/sigstore/sigstore v1.2.1-0.20220424143412-3d41663116d5
	github.com/spf13/cobra v1.4.0
	github.com/spf13/viper v1.11.0
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613
	github.com/theupdateframework/go-tuf v0.0.0-20220211205608-f0c3294f63b9
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

replace github.com/theupdateframework/go-tuf => github.com/asraa/go-tuf v0.0.0-20211118155909-342063f69dee
