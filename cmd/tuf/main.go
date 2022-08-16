//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build pivkey
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
