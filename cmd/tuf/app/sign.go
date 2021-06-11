// +build pivkey

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
	cjson "github.com/tent/canonical-json-go"
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

func CreateDb(store tuf.LocalStore) (*verify.DB, error) {
	db := verify.NewDB()
	root, err := GetRootFromStore(store)
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
	db, err := CreateDb(store)
	if err != nil {
		return fmt.Errorf("error creating verification database: %w", err)
	}
	for _, role := range role {
		switch role {
		case "snapshot":
			// Check that root and target are signed correctly
			for _, manifest := range []string{"root", "targets"} {
				s, err := GetSignedMeta(store, manifest+".json")
				if err != nil {
					return err
				}

				if err := db.Verify(s, manifest, 0); err != nil {
					return fmt.Errorf("error verifying signatures for %s: %w", manifest, err)
				}
			}
			// At this point we have valid root and targets. Maybe update hashes,
			if err := updateSnapshot(store); err != nil {
				return err
			}
		case "timestamp":
			// Check that snapshot is signed
			s, err := GetSignedMeta(store, "snapshot.json")
			if err != nil {
				return err
			}
			if err := db.Verify(s, "snapshot", 0); err != nil {
				return fmt.Errorf("error verifying signatures for snapshot: %w", err)
			}
			// At this point we have a valid snapshot.
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
	// This assumes the root.json and targets.json are correctly signed.
	// Check the snapshot metadata
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

	changed := false
	for _, name := range []string{"root.json", "targets.json"} {
		b := meta[name]
		meta, err := util.GenerateSnapshotFileMeta(bytes.NewReader(b))
		if err != nil {
			return err
		}

		if prev, ok := snapshot.Meta[name]; !ok || util.SnapshotFileMetaEqual(prev, meta) != nil {
			snapshot.Meta[name] = meta
			changed = true
		}
	}

	if changed {
		// We only setMeta if we needed to change the root or targets hashes. Otherwise, this
		// removes old signatures.
		// Keep the initial set of empty sigs
		signed, err := jsonMarshal(snapshot)
		if err != nil {
			return err
		}
		s.Signed = signed
		return setSignedMeta(store, "snapshot.json", s)
	}
	return nil
}

func updateTimestamp(store tuf.LocalStore) error {
	s, err := GetSignedMeta(store, "timestamp.json")
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

	fileMeta, err := util.GenerateTimestampFileMeta(bytes.NewReader(b))
	if err != nil {
		return err
	}

	if prev, ok := timestamp.Meta["snapshot.json"]; !ok || util.TimestampFileMetaEqual(prev, fileMeta) != nil {
		// We only setMeta if we needed to change the root or targets hashes. Otherwise, this
		// removes old signatures.
		timestamp.Meta["snapshot.json"] = fileMeta
		signed, err := jsonMarshal(timestamp)
		if err != nil {
			return err
		}
		s.Signed = signed
		return setSignedMeta(store, "timestamp.json", s)
	}

	return nil
}

func SignMeta(ctx context.Context, store tuf.LocalStore, name string, signer signature.Signer, key *data.Key) error {
	fmt.Printf("Signing metadata for %s... \n", name)
	s, err := GetSignedMeta(store, name)
	if err != nil {
		return err
	}

	// Sign payload
	var decoded map[string]interface{}
	if err := json.Unmarshal(s.Signed, &decoded); err != nil {
		return err
	}
	msg, err := cjson.Marshal(decoded)
	if err != nil {
		return err
	}

	sig, _, err := signer.Sign(ctx, msg)
	if err != nil {
		return err
	}

	if s.Signatures == nil {
		// init-repo should have pre-populated these. don't lose them.
		return errors.New("pre-entries not defined")
	}
	sigs := make([]data.Signature, 0, len(s.Signatures))

	// Add it to your key entry
	for _, id := range key.IDs() {
		for _, entry := range s.Signatures {
			if entry.KeyID == id {
				sigs = append(sigs, data.Signature{
					KeyID:     id,
					Signature: sig,
				})
			} else {
				sigs = append(sigs, entry)
			}
		}
	}

	return setSignedMeta(store, name, &data.Signed{Signatures: sigs, Signed: s.Signed})
}
