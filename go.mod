module github.com/sigstore/root-signing

go 1.16

require (
	github.com/peterbourgon/ff/v3 v3.1.0
	github.com/pkg/errors v0.9.1
	github.com/sigstore/cosign v1.4.1
	github.com/sigstore/sigstore v1.0.2-0.20211203233310-c8e7f70eab4e
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613
	github.com/theupdateframework/go-tuf v0.0.0-20211209174453-13f0687177ba
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

replace github.com/theupdateframework/go-tuf => github.com/asraa/go-tuf v0.0.0-20211118155909-342063f69dee
