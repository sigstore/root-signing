package app

import (
	"context"
	"encoding/json"
	"flag"

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
	m, err := store.GetMeta()
	if err != nil {
		return err
	}
	// If snapshotting fails, restore the original state of the store
	// before clearing the signatures.
	var snapshotReturnErr error
	defer func(m map[string]json.RawMessage) {
		if snapshotReturnErr != nil {
			for mname, mdata := range m {
				if err := store.SetMeta(mname, mdata); err != nil {
					// Set the return error.
					snapshotReturnErr = err
				}
			}
		}
	}(m)

	// Clear any empty placeholder signatures.
	for _, metaname := range []string{"root.json", "targets.json"} {
		if err := ClearEmptySignatures(store, metaname); err != nil {
			return err
		}
	}

	repo, err := tuf.NewRepoIndent(store, "", "\t", "sha512", "sha256")
	if err != nil {
		return err
	}

	snapshotReturnErr = repo.SnapshotWithExpires(getExpiration("snapshot"))
	return snapshotReturnErr
}
