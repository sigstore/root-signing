package app

import (
	"context"
	"flag"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/theupdateframework/go-tuf"
)

func Snapshot() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf snapshot", flag.ExitOnError)
		repository = flagset.String("repository", "", "path to the staged repository")
	)
	return &ffcli.Command{
		Name:       "snapshot",
		ShortUsage: "tuf snapshot the top-level metadata in the given repository",
		ShortHelp:  "tuf snapshot the top-level metadata in the given repository",
		LongHelp: `tuf snapshot the top-level metadata in the given repository.
		It adds them to all four top-level roles. 
		
	EXAMPLES
	# snapshot staged repository at ceremony/YYYY-MM-DD
	tuf snapshot -repository ceremony/YYYY-MM-DD`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *repository == "" {
				return flag.ErrHelp
			}
			return SnapshotCmd(ctx, *repository)
		},
	}
}

func SnapshotCmd(ctx context.Context, directory string) error {
	store := tuf.FileSystemStore(directory, nil)

	repo, err := tuf.NewRepoIndent(store, "", "\t")
	if err != nil {
		return err
	}
	return repo.SnapshotWithExpires(time.Now().AddDate(0, 0, 14).UTC())
}
