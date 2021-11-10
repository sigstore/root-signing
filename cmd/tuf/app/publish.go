package app

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/theupdateframework/go-tuf"
)

// Ensure all versions are 1
// Ensure all signatures are correct and correct number
// Publish by moving to final_dir

func Publish() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf publish", flag.ExitOnError)
		repository = flagset.String("repository", "", "path to the staged repository")
	)
	return &ffcli.Command{
		Name:       "publish",
		ShortUsage: "tuf publish the top-level metadata in the given repository",
		ShortHelp:  "tuf publish the top-level metadata in the given repository",
		LongHelp: `tuf publish the top-level metadata in the given repository.
		It adds them to all four top-level roles. 
		
	EXAMPLES
	# publish staged repository at ceremony/YYYY-MM-DD
	tuf publish -repository ceremony/YYYY-MM-DD`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *repository == "" {
				return flag.ErrHelp
			}
			return PublishCmd(ctx, *repository)
		},
	}
}

func PublishCmd(ctx context.Context, directory string) error {
	store := tuf.FileSystemStore(directory, nil)

	repo, err := tuf.NewRepoIndent(store, "", "\t", "sha512", "sha256")
	if err != nil {
		return err
	}
	if err := repo.Commit(); err != nil {
		return err
	}

	fmt.Println("Metadata successfully validated!")
	return nil
}
