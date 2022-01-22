module github.com/sigstore/root-signing

go 1.16

require (
	github.com/peterbourgon/ff/v3 v3.1.2
	github.com/pkg/errors v0.9.1
	github.com/sigstore/cosign v1.5.0
	github.com/sigstore/sigstore v1.1.1-0.20220115165716-9f61ddc98390
	github.com/tent/canonical-json-go v0.0.0-20130607151641-96e4ba3a7613
	github.com/theupdateframework/go-tuf v0.0.0-20220113233521-eac0a85ce281
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

replace github.com/theupdateframework/go-tuf => github.com/asraa/go-tuf v0.0.0-20211118155909-342063f69dee
