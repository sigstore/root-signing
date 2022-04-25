//go:build pivkey
// +build pivkey

package app

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sigstore/cosign/pkg/cosign/pivkey"
	"github.com/sigstore/root-signing/pkg/keys"
	"github.com/sigstore/root-signing/pkg/repo"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/options"
	cjson "github.com/tent/canonical-json-go"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
)

type roleFlag []string

func (f *roleFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *roleFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func Sign() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf sign", flag.ExitOnError)
		roles      = roleFlag{}
		repository = flagset.String("repository", "", "path to the staged repository")
		sk         = flagset.Bool("sk", false, "indicates use of a hardware key for signing")
		key        = flagset.String("key", "", "reference to an onine signer for signing")
	)
	flagset.Var(&roles, "roles", "role(s) to sign")
	return &ffcli.Command{
		Name:       "sign",
		ShortUsage: "tuf signs the top-level metadata for role in the given repository",
		ShortHelp:  "tuf signs the top-level metadata for role in the given repository",
		LongHelp: `tuf signs the top-level metadata for role in the given repository.
		Signing a lower level, e.g. snapshot or timestamp, before signing the root and target
		will trigger a warning. 
		One of sk or a key reference must be provided.
		
	EXAMPLES
	# sign staged repository at ceremony/YYYY-MM-DD
	tuf sign -role root -repository ceremony/YYYY-MM-DD`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *repository == "" || len(roles) == 0 {
				return flag.ErrHelp
			}
			if !*sk && *key == "" {
				return flag.ErrHelp
			}
			signerAndKey, err := getSigner(ctx, *sk, *key)
			if err != nil {
				return err
			}
			return SignCmd(ctx, *repository, roles, signerAndKey)
		},
	}
}

func checkMetaForRole(store tuf.LocalStore, role []string) error {
	db, _, err := repo.CreateDb(store)
	if err != nil {
		return fmt.Errorf("error creating verification database: %w", err)
	}
	for _, role := range role {
		switch role {
		case "snapshot":
			// Check that root and target are signed correctly
			for _, manifest := range []string{"root", "targets"} {
				s, err := repo.GetSignedMeta(store, manifest+".json")
				if err != nil {
					return err
				}

				if err := db.Verify(s, manifest, 0); err != nil {
					return fmt.Errorf("error verifying signatures for %s: %w", manifest, err)
				}
			}
		case "timestamp":
			// Check that snapshot is signed
			s, err := repo.GetSignedMeta(store, "snapshot.json")
			if err != nil {
				return err
			}
			if err := db.Verify(s, "snapshot", 0); err != nil {
				return fmt.Errorf("error verifying signatures for snapshot: %w", err)
			}
		case "default":
			// No pre-requisites for signing root and target
			continue
		}
	}
	return nil
}

func getSigner(ctx context.Context, sk bool, keyRef string) (*keys.SignerAndTufKey, error) {
	if sk {
		// This will give us the data.PublicKey with the correct id.
		keyAndAttestations, err := GetKeyAndAttestation(ctx)
		if err != nil {
			return nil, err
		}
		pivKey, err := pivkey.GetKeyWithSlot("signature")
		if err != nil {
			return nil, err
		}
		signer, err := pivKey.SignerVerifier()
		if err != nil {
			return nil, err
		}
		return &keys.SignerAndTufKey{Signer: signer, Key: keyAndAttestations.key}, nil
	}
	// A key reference was provided.
	return keys.GetKmsSigningKey(ctx, keyRef)
}

func SignCmd(ctx context.Context, directory string, roles []string, signer *keys.SignerAndTufKey) error {
	store := tuf.FileSystemStore(directory, nil)

	if err := checkMetaForRole(store, roles); err != nil {
		return fmt.Errorf("signing pre-requisites failed: %w", err)
	}

	for _, name := range roles {
		if err := SignMeta(ctx, store, name+".json", signer.Signer, signer.Key); err != nil {
			return err
		}
	}

	return nil
}

func SignMeta(ctx context.Context, store tuf.LocalStore, name string, signer signature.Signer, key *data.PublicKey) error {
	fmt.Printf("Signing metadata for %s... \n", name)
	s, err := repo.GetSignedMeta(store, name)
	if err != nil {
		return err
	}
	if (name == "root.json" || name == "targets.json") && s.Signatures == nil {
		// init-repo should have pre-populated these. don't lose them.
		return errors.New("pre-entries not defined")
	}

	// Sign payload
	meta, err := repo.GetMetaFromStore(s.Signed, name)
	if err != nil {
		return err
	}
	msg, err := cjson.Marshal(meta)
	if err != nil {
		return err
	}

	sig, err := signer.SignMessage(bytes.NewReader(msg), options.WithContext(ctx))
	if err != nil {
		return err
	}

	sigs := make([]data.Signature, 0, len(s.Signatures))

	// Add it to your key entry
	for _, id := range key.IDs() {
		// If pre-entries are defined.
		if s.Signatures != nil {
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
		} else {
			sigs = append(sigs, data.Signature{
				KeyID:     id,
				Signature: sig,
			})
		}
	}

	return setSignedMeta(store, name, &data.Signed{Signatures: sigs, Signed: s.Signed})
}
