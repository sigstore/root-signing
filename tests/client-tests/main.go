//
// Copyright 2022 The Sigstore Authors.
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
//
// NOTE: This is mostly a copy of the go-tuf client, but with deprecated ECSDA
// support in order to provide full verification of old and new formatted keys.
// See https://github.com/theupdateframework/go-tuf/blob/master/cmd/tuf-client

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"text/tabwriter"

	"github.com/dustin/go-humanize"
	docopt "github.com/flynn/go-docopt"
	tuf "github.com/theupdateframework/go-tuf/client"
	tuf_leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"
	_ "github.com/theupdateframework/go-tuf/pkg/deprecated/set_ecdsa"
)

func main() {
	log.SetFlags(0)

	usage := `usage: tuf-client [-h|--help] <command> [<args>...]
Options:
  -h, --help
Commands:
  help         Show usage for a specific command
  init         Initialize with root keys
  list         List available target files
  get          Get a target file
See "tuf-client help <command>" for more information on a specific command.
`
	register("init", cmdInit, `
usage: tuf-client init [-s|--store=<path>] <url> [<root-metadata-file>]
Options:
  -s <path>    The path to the local file store [default: tuf.db]
Initialize the local file store with root metadata.
  `)

	register("list", cmdList, `
usage: tuf-client list [-s|--store=<path>] <url>
Options:
  -s <path>    The path to the local file store [default: tuf.db]
List available target files.
	`)

	args, _ := docopt.Parse(usage, nil, true, "", true)
	cmd := args.String["<command>"]
	cmdArgs := args.All["<args>"].([]string)

	if cmd == "help" {
		if len(cmdArgs) == 0 { // `tuf-client help`
			fmt.Fprint(os.Stdout, usage)
			return
		}
		// `tuf-client help <command>`
		cmd = cmdArgs[0]
		cmdArgs = []string{"--help"}
	}

	if err := runCommand(cmd, cmdArgs); err != nil {
		log.Fatalln("ERROR:", err)
	}
}

type cmdFunc func(*docopt.Args, *tuf.Client) error

type command struct {
	usage string
	f     cmdFunc
}

var commands = make(map[string]*command)

func register(name string, f cmdFunc, usage string) {
	commands[name] = &command{usage: usage, f: f}
}

func runCommand(name string, args []string) error {
	argv := make([]string, 1, 1+len(args))
	argv[0] = name
	argv = append(argv, args...)

	cmd, ok := commands[name]
	if !ok {
		return fmt.Errorf("%s is not a tuf-client command. See 'tuf-client help'", name)
	}

	parsedArgs, err := docopt.Parse(cmd.usage, argv, true, "", true)
	if err != nil {
		return err
	}

	client, err := tufClient(parsedArgs)
	if err != nil {
		return err
	}
	return cmd.f(parsedArgs, client)
}

func tufClient(args *docopt.Args) (*tuf.Client, error) {
	store, ok := args.String["--store"]
	if !ok {
		store = args.String["-s"]
	}
	local, err := tuf_leveldbstore.FileLocalStore(store)
	if err != nil {
		return nil, err
	}
	remote, err := tuf.HTTPRemoteStore(args.String["<url>"], nil, nil)
	if err != nil {
		return nil, err
	}
	return tuf.NewClient(local, remote), nil
}

func cmdInit(args *docopt.Args, client *tuf.Client) error {
	file := args.String["<root-metadata-file>"]
	var in io.Reader
	if file == "" || file == "-" {
		in = os.Stdin
	} else {
		var err error
		in, err = os.Open(file)
		if err != nil {
			return err
		}
	}
	bytes, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	return client.Init(bytes)
}

func cmdList(args *docopt.Args, client *tuf.Client) error {
	if _, err := client.Update(); err != nil {
		return err
	}
	targets, err := client.Targets()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "PATH\tSIZE")
	for path, meta := range targets {
		fmt.Fprintf(w, "%s\t%s\n", path, humanize.Bytes(uint64(meta.Length)))
	}
	return nil
}
