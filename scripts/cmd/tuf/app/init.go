package app

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	cjson "github.com/tent/canonical-json-go"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/util"
)

var threshold = 3

type targetsFlag []string

func (f *targetsFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *targetsFlag) Set(value string) error {
	if _, err := os.Stat(filepath.Clean(value)); os.IsNotExist(err) {
		return err
	}
	*f = append(*f, value)
	return nil
}

func Init() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf init", flag.ExitOnError)
		repository = flagset.String("repository", "", "path to initialize the staged repository")
		targets    = targetsFlag{}
	)
	flagset.Var(&targets, "target", "path to target to add")
	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "tuf init initializes a new TUF repository",
		ShortHelp:  "tuf init initializes a new TUF repository",
		LongHelp: `tuf init initializes a new TUF repository to the 
		specified repository directory. It will create unpopulated directories 
		keys/, staged/, and staged/targets under the repository with threshold 3
		and a 4 month expiration.
		
	EXAMPLES
	# initialize repository at ceremony/YYYY-MM-DD
	tuf init -repository ceremony/YYYY-MM-DD`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *repository == "" {
				return flag.ErrHelp
			}
			return InitCmd(*repository, targets)
		},
	}
}

func InitCmd(directory string, targets targetsFlag) error {
	// TODO: Validate directory is a good path.
	store := tuf.FileSystemStore(directory, nil)
	repo, err := tuf.NewRepo(store)
	if err != nil {
		return err
	}
	if err := repo.Init(false); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "TUF repository initialized at ", directory)

	// Add top-level targets with threshold and expiration.
	expiration := time.Now().AddDate(0, 4, 0)
	relativePaths := []string{}
	if err != nil {
		return err
	}
	for _, target := range targets {
		// Copy target into directory/staged/targets
		from, err := os.Open(target)
		if err != nil {
			return err
		}
		defer from.Close()
		base := filepath.Base(target)
		to, err := os.OpenFile(filepath.Join(directory, "staged/targets", base), os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		defer to.Close()
		if _, err := io.Copy(to, from); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Created target file at ", to.Name())
		relativePaths = append(relativePaths, base)
	}

	// TODO: go-tuf does not support customizing root expiration. Hacked.
	// We very hackishly by unmarshalling the JSON and do it manually.
	root, err := getRootFromStore(store)
	if err != nil {
		return err
	}
	root.Expires = expiration
	// TODO: go-tuf also does not support modifying thresholds.
	roles := []string{"root", "targets", "timestamp", "snapshot"}
	for _, roleName := range roles {
		_, ok := root.Roles[roleName]
		if !ok {
			role := &data.Role{KeyIDs: []string{}, Threshold: threshold}
			root.Roles[roleName] = role
		}
	}

	// After all this hack-ing, set the files with the updated root.
	setMeta(store, "root.json", root)

	// Add targets.
	if err := repo.AddTargetsWithExpires(relativePaths, nil, expiration); err != nil {
		return fmt.Errorf("error adding targets %w", err)
	}

	// Create snapshot.json
	snapshot, err := createNewSnapshot(store, expiration)
	if err != nil {
		return err
	}
	setMeta(store, "snapshot.json", snapshot)

	// Create timestamp.json
	timestamp, err := createNewTimestamp(store, expiration)
	if err != nil {
		return err
	}
	setMeta(store, "timestamp.json", timestamp)

	return nil
}

func createNewSnapshot(store tuf.LocalStore, expires time.Time) (*data.Snapshot, error) {
	snapshot := data.NewSnapshot()
	snapshot.Expires = expires
	// The go implementation also includes root.json
	meta, err := store.GetMeta()
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"root.json", "targets.json"} {
		b := meta[name]
		fileMeta, err := util.GenerateSnapshotFileMeta(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		snapshot.Meta[name] = fileMeta
	}
	return snapshot, nil
}

func createNewTimestamp(store tuf.LocalStore, expires time.Time) (*data.Timestamp, error) {
	timestamp := data.NewTimestamp()
	timestamp.Expires = expires

	meta, err := store.GetMeta()
	if err != nil {
		return nil, err
	}
	b := meta["snapshot.json"]
	fileMeta, err := util.GenerateTimestampFileMeta(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	timestamp.Meta["snapshot.json"] = fileMeta

	return timestamp, nil
}

func getSignedMeta(store tuf.LocalStore, name string) (*data.Signed, error) {
	// Get name signed meta (name is of the form *.json)
	meta, err := store.GetMeta()
	if err != nil {
		return nil, err
	}
	signedJSON, ok := meta[name]
	if !ok {
		return nil, fmt.Errorf("missing metadata %s", name)
	}
	s := &data.Signed{}
	if err := json.Unmarshal(signedJSON, s); err != nil {
		return nil, err
	}
	return s, nil
}

func getRootFromStore(store tuf.LocalStore) (*data.Root, error) {
	s, err := getSignedMeta(store, "root.json")
	if err != nil {
		return nil, err
	}
	root := &data.Root{}
	if err := json.Unmarshal(s.Signed, root); err != nil {
		return nil, err
	}
	return root, nil
}

func setMeta(store tuf.LocalStore, role string, meta interface{}) error {
	signed, err := jsonMarshal(meta)
	if err != nil {
		return err
	}

	return setSignedMeta(store, role, &data.Signed{Signed: signed})
}

func setSignedMeta(store tuf.LocalStore, role string, s *data.Signed) error {
	b, err := jsonMarshal(s)
	if err != nil {
		return err
	}
	return store.SetMeta(role, b)
}

func jsonMarshal(v interface{}) ([]byte, error) {
	signed, err := cjson.Marshal(v)
	if err != nil {
		return nil, err
	}
	return signed, nil
}
