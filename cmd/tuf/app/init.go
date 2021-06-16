// +build pivkey

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	pkeys "github.com/sigstore/root-signing/pkg/keys"
	prepo "github.com/sigstore/root-signing/pkg/repo"
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

	// Get the root.json file and initialize it with the expirations and thresholds
	root, err := prepo.GetRootFromStore(store)
	if err != nil {
		return err
	}
	expiration := time.Now().AddDate(0, 6, 0)
	relativePaths := []string{}
	root.Expires = expiration
	root.Version++
	// Add the keys we just provisioned to each role
	keys, err := getKeysFromDir(directory + "/keys")
	if err != nil {
		return err
	}

	roles := []string{"root", "targets", "timestamp", "snapshot"}
	for _, tufKey := range keys {
		root.AddKey(tufKey)
		for _, roleName := range roles {
			role, ok := root.Roles[roleName]
			if !ok {
				role = &data.Role{KeyIDs: []string{}, Threshold: threshold}
				root.Roles[roleName] = role
			}
			role.AddKeyIDs(tufKey.IDs())
		}
	}

	// After all this root metadata setting, set the files with the updated root
	if err := setMetaWithSigKeyIDs(store, "root.json", root, keys); err != nil {
		return err
	}

	// Add targets (copy them into the repository and add them to the targets.json)
	for _, target := range targets {
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

	if err := repo.AddTargetsWithExpires(relativePaths, nil, expiration); err != nil {
		return fmt.Errorf("error adding targets %w", err)
	}

	// Add blank signatures
	t, err := prepo.GetTargetsFromStore(store)
	if err != nil {
		return err
	}
	t.Version = root.Version
	if err := setMetaWithSigKeyIDs(store, "targets.json", t, keys); err != nil {
		return err
	}

	// Create snapshot.json
	snapshot, err := createNewSnapshot(store, expiration)
	if err != nil {
		return err
	}
	snapshot.Version = root.Version
	if err := setMetaWithSigKeyIDs(store, "snapshot.json", snapshot, keys); err != nil {
		return err
	}

	// Create timestamp.json
	timestamp, err := createNewTimestamp(store, expiration)
	if err != nil {
		return err
	}
	timestamp.Version = root.Version
	return setMetaWithSigKeyIDs(store, "timestamp.json", timestamp, keys)
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

func setSignedMeta(store tuf.LocalStore, role string, s *data.Signed) error {
	b, err := jsonMarshal(s)
	if err != nil {
		return err
	}
	return store.SetMeta(role, b)
}

func setMetaWithSigKeyIDs(store tuf.LocalStore, role string, meta interface{}, keys []*data.Key) error {
	signed, err := jsonMarshal(meta)
	if err != nil {
		return err
	}

	// Add empty sigs
	emptySigs := make([]data.Signature, 0, 1)

	for _, key := range keys {
		for _, id := range key.IDs() {
			emptySigs = append(emptySigs, data.Signature{
				KeyID: id,
			})
		}
	}

	return setSignedMeta(store, role, &data.Signed{Signatures: emptySigs, Signed: signed})
}

func jsonMarshal(v interface{}) ([]byte, error) {
	b, err := cjson.Marshal(v)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "\t"); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func getKeysFromDir(dir string) ([]*data.Key, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var tufKeys []*data.Key
	for _, file := range files {
		if file.IsDir() {
			key, err := pkeys.SigningKeyFromDir(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			tufKeys = append(tufKeys, pkeys.ToTufKey(*key))
		}
	}
	return tufKeys, err
}
