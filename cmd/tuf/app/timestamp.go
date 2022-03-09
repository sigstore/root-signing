package app

import (
	"context"
	"flag"
	"time"

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
	return repo.TimestampWithExpires(time.Now().AddDate(0, 0, timestampDaysExpires).UTC())
}
