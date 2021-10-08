// +build pivkey

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sigstore/root-signing/cmd/tuf/app"
)

var (
	rootFlagSet = flag.NewFlagSet("tuf", flag.ExitOnError)
)

func main() {
	root := &ffcli.Command{
		ShortUsage:  "tuf [flags] <subcommand>",
		FlagSet:     rootFlagSet,
		Subcommands: []*ffcli.Command{app.Init(), app.AddKey(), app.Snapshot(), app.AddDelegation(), app.Timestamp(), app.Sign(), app.Publish()},
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}

	if err := root.Parse(os.Args[1:]); err != nil {
		printErrAndExit(err)
	}

	if err := root.Run(context.Background()); err != nil {
		printErrAndExit(err)
	}
}

func printErrAndExit(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
