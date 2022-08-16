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

package app

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/theupdateframework/go-tuf"
)

func Timestamp() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf timestamp", flag.ExitOnError)
		repository = flagset.String("repository", "", "path to the staged repository")
	)
	return &ffcli.Command{
		Name:       "timestamp",
		ShortUsage: "tuf timestamp the top-level metadata in the given repository",
		ShortHelp:  "tuf timestamp the top-level metadata in the given repository",
		LongHelp: `tuf timestamp the top-level metadata in the given repository.
		It adds them to all four top-level roles. 
		
	EXAMPLES
	# timestamp staged repository at ceremony/YYYY-MM-DD
	tuf timestamp -repository ceremony/YYYY-MM-DD`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *repository == "" {
				return flag.ErrHelp
			}
			return TimestampCmd(ctx, *repository)
		},
	}
}

func TimestampCmd(ctx context.Context, directory string) error {
	store := tuf.FileSystemStore(directory, nil)

	repo, err := tuf.NewRepoIndent(store, "", "\t", "sha512", "sha256")
	if err != nil {
		return err
	}
	return repo.TimestampWithExpires(getExpiration("timestamp"))
}
