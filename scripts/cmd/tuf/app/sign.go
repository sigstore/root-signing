package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sigstore/cosign/pkg/cosign/pivkey"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/util"
	"github.com/theupdateframework/go-tuf/verify"
)

var roles = map[string]bool{"root": true, "targets": true, "timestamp": true, "snapshot": true}

type roleFlag []string

func (f *roleFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *roleFlag) Set(value string) error {
	if !roles[value] {
		return fmt.Errorf("%s not a valid role", value)
	}
	*f = append(*f, value)
	return nil
}

func Sign() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf sign", flag.ExitOnError)
		roles      = roleFlag{}
		repository = flagset.String("repository", "", "path to the staged repository")
	)
	flagset.Var(&roles, "roles", "role(s) to sign")
	return &ffcli.Command{
		Name:       "sign",
		ShortUsage: "tuf signs the top-level metadata for role in the given repository",
		ShortHelp:  "tuf signs the top-level metadata for role in the given repository",
		LongHelp: `tuf signs the top-level metadata for role in the given repository.
		Signing a lower level, e.g. snapshot or timestamp, before signing the root and target
		will trigger a warning. 
		
	EXAMPLES
	# sign staged repository at ceremony/YYYY-MM-DD
	tuf sign -role root -repository ceremony/YYYY-MM-DD`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *repository == "" || len(roles) == 0 {
				return flag.ErrHelp
			}
			return SignCmd(ctx, *repository, roles)
		},
	}
}

func createDb(store tuf.LocalStore) (*verify.DB, error) {
	db := verify.NewDB()
	root, err := getRootFromStore(store)
	if err != nil {
		return nil, err
	}
	for id, k := range root.Keys {
		if err := db.AddKey(id, k); err != nil {
			// ignore ErrWrongID errors by TAP-12
			if _, ok := err.(verify.ErrWrongID); !ok {
				return nil, err
			}
		}
	}
	for name, role := range root.Roles {
		if err := db.AddRole(name, role); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func checkAndUpdateMetaForRole(store tuf.LocalStore, role []string) error {
	db, err := createDb(store)
	if err != nil {
		return fmt.Errorf("error creating verification database: %w", err)
	}
	for _, role := range role {
		switch role {
		case "snapshot":
			// Check that root and target are signed correctly
			for _, manifest := range []string{"root", "targets"} {
				s, err := getSignedMeta(store, manifest+".json")
				if err != nil {
					return err
				}
				if err := db.Verify(s, manifest, 0); err != nil {
					return fmt.Errorf("error verifying signatures for %s", manifest)
				}
			}

			if err := updateSnapshot(store); err != nil {
				return err
			}
		case "timestamp":
			// Check that snapshot is signed
			s, err := getSignedMeta(store, "snapshot.json")
			if err != nil {
				return err
			}
			if err := db.Verify(s, "snapshot", 0); err != nil {
				return errors.New("error verifying signatures for snapshot")
			}

			if err := updateTimestamp(store); err != nil {
				return err
			}
		case "default":
			// No pre-requisites for signing root and target
			continue
		}
	}
	return nil
}

func SignCmd(ctx context.Context, directory string, roles []string) error {
	store := tuf.FileSystemStore(directory, nil)

	if err := checkAndUpdateMetaForRole(store, roles); err != nil {
		return fmt.Errorf("signing pre-requisites failed: %w", err)
	}

	// This will give us the data.Key with the correct id.
	keyAndAttestations, err := GetKeyAndAttestation(ctx)
	if err != nil {
		return err
	}

	signer, err := pivkey.NewSignerVerifier()
	if err != nil {
		return err
	}

	for _, name := range roles {
		if err := SignMeta(ctx, store, name+".json", signer, keyAndAttestations.key); err != nil {
			return err
		}
	}

	return nil
}

func updateSnapshot(store tuf.LocalStore) error {
	meta, err := store.GetMeta()
	if err != nil {
		return err
	}
	snapshotJSON := meta["snapshot.json"]
	s := &data.Signed{}
	if err := json.Unmarshal(snapshotJSON, s); err != nil {
		return err
	}
	snapshot := &data.Snapshot{}
	if err := json.Unmarshal(s.Signed, snapshot); err != nil {
		return err
	}

	for _, name := range []string{"root.json", "targets.json"} {
		b := meta[name]
		snapshot.Meta[name], err = util.GenerateSnapshotFileMeta(bytes.NewReader(b))
		if err != nil {
			return err
		}
	}

	return setMeta(store, "snapshot.json", snapshot)
}

func updateTimestamp(store tuf.LocalStore) error {
	s, err := getSignedMeta(store, "timestamp.json")
	if err != nil {
		return err
	}
	timestamp := &data.Timestamp{}
	if err := json.Unmarshal(s.Signed, timestamp); err != nil {
		return err
	}

	meta, err := store.GetMeta()
	if err != nil {
		return err
	}
	b, ok := meta["snapshot.json"]
	if !ok {
		return errors.New("missing metadata: snapshot.json")
	}
	timestamp.Meta["snapshot.json"], err = util.GenerateTimestampFileMeta(bytes.NewReader(b))
	if err != nil {
		return err
	}

	return setMeta(store, "timestamp.json", timestamp)
}

func SignMeta(ctx context.Context, store tuf.LocalStore, name string, signer signature.Signer, key *data.Key) error {
	fmt.Printf("Signing metadata for %s... \n", name)
	s, err := getSignedMeta(store, name)
	if err != nil {
		return err
	}

	// Sign payload
	sig, _, err := signer.Sign(ctx, s.Signed)
	if err != nil {
		return err
	}
	if s.Signatures == nil {
		s.Signatures = make([]data.Signature, 0, 1)
	}
	for _, id := range key.IDs() {
		s.Signatures = append(s.Signatures, data.Signature{
			KeyID:     id,
			Signature: sig,
		})
	}

	return setSignedMeta(store, name, s)
}
