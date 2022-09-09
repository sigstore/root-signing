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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	pkeys "github.com/sigstore/root-signing/pkg/keys"
	prepo "github.com/sigstore/root-signing/pkg/repo"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
)

// Threshold for root and targets signers.
var DefaultThreshold = 3

// Enable consistent snapshotting.
var ConsistentSnapshot = true

// Time to role expiration represented as a list of ints corresponding to
// (years, months, days).
var RoleExpiration = map[string][]int{
	"root":      {0, 6, 0},
	"targets":   {0, 6, 0},
	"snapshot":  {0, 0, 21},
	"timestamp": {0, 0, 14},
}

func getExpiration(role string) time.Time {
	// Default expiration for any delegated role is the targets expiration.
	times, ok := RoleExpiration[role]
	if !ok {
		times = RoleExpiration["targets"]
		fmt.Fprintf(os.Stderr, "Explicit expiration not found, using default targets expiration in %d years, %d months, %d days\n",
			times[0], times[1], times[2])
	}
	return time.Now().AddDate(times[0], times[1], times[2]).UTC().Round(time.Second)
}

func Init() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf init", flag.ExitOnError)
		repository = flagset.String("repository", "", "path to initialize the staged repository")
		threshold  = flagset.Int("threshold", DefaultThreshold, "default root and targets signer threshold")
		previous   = flagset.String("previous", "", "path to the previous repository")
		snapshot   = flagset.String("snapshot", "", "reference to an online snapshot signer")
		timestamp  = flagset.String("timestamp", "", "reference to an online timestamp signer")
		targets    = flagset.String("target-meta", "", "path to a target configuration file")
	)
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
			if *snapshot == "" {
				return flag.ErrHelp
			}
			if *timestamp == "" {
				return flag.ErrHelp
			}
			if *targets == "" {
				return flag.ErrHelp
			}
			targetsConfigStr, err := os.ReadFile(*targets)
			if err != nil {
				return err
			}
			targetsConfig, err := prepo.SigstoreTargetMetaFromString(targetsConfigStr)
			if err != nil {
				return err
			}
			return InitCmd(ctx, *repository, *previous, *threshold, targetsConfig, *snapshot, *timestamp)
		},
	}
}

// InitCmd creates a new staged root.json and targets.json in the specified directory. It populates the top-level
// roles with signers and adds targets to top-level targets.
//   - directory: Directory to write newly staged metadata. Must contain a keys/ subdirectory with root/targets signers.
//   - previous: Optional previous repository to chain the root from.
//   - threshold: The root and targets threshold.
//   - targetsConfig: A map of target file names and custom metadata to add to top-level targets.
//     Target file names are expected to be in the working directory.
//   - snapshotRef: A reference (KMS, file, URL) to a snapshot signer.
//   - timestampRef: A reference (KMS, file, URL) to a timestamp signer.
//
// The root and targets metadata will be initialized with a 6 month expiration.
// Revoked keys will be automatically calculated given the previous root and the signers in directory.
// Signature placeholders for each key will be added to the root.json and targets.json file.
func InitCmd(ctx context.Context, directory, previous string, threshold int, targetsConfig map[string]json.RawMessage, snapshotRef string, timestampRef string) error {
	// TODO: Validate directory is a good path.
	store := tuf.FileSystemStore(directory, nil)
	repo, err := tuf.NewRepoIndent(store, "", "\t", "sha512", "sha256")
	if err != nil {
		return err
	}
	curRootVersion, err := repo.RootVersion()
	if err != nil {
		return err
	}

	if previous == "" {
		// Only initialize if no previous specified.
		if err := repo.Init(ConsistentSnapshot); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "TUF repository initialized at ", directory)
	}

	// Add the keys we just provisioned to root and targets and revoke any removed ones.
	root, err := prepo.GetRootFromStore(store)
	if err != nil {
		return err
	}
	keys, err := getKeysFromDir(directory + "/keys")
	if err != nil {
		return err
	}
	var allRootKeys []*data.PublicKey
	// Add any keys in the keys/ subfolder to root and targets.
	for _, role := range []string{"root", "targets"} {
		currentKeyMap := map[string]bool{}
		for _, tufKey := range keys {
			currentKeyMap[tufKey.IDs()[0]] = true
			if err := repo.AddVerificationKeyWithExpiration(role, tufKey, getExpiration(role)); err != nil {
				return err
			}
		}
		if role == "root" {
			// This retrieves all the new root keys, but before we revoke any.
			// This is used to populate the placeholder signature key IDs, which is composed
			// of the (old keys + current keys)
			allRootKeys, err = repo.RootKeys()
			if err != nil {
				return err
			}
		}
		// Revoke any keys that we've rotated out
		oldKeys, ok := root.Roles[role]
		if ok {
			for _, oldKeyID := range oldKeys.KeyIDs {
				if _, ok := currentKeyMap[oldKeyID]; !ok {
					if err := repo.RevokeKey(role, oldKeyID); err != nil {
						return err
					}
				}
			}
		}
		if err := repo.SetThreshold(role, threshold); err != nil {
			return err
		}
	}

	// Add keys used for snapshot and timestamp roles.
	for role, keyRef := range map[string]string{"snapshot": snapshotRef, "timestamp": timestampRef} {
		// Snapshot and Timestamp roles may use the new PEM format!
		signerKey, err := pkeys.GetSigningKey(ctx, keyRef, false)
		if err != nil {
			return err
		}

		// Add key.
		if err := repo.AddVerificationKeyWithExpiration(role, signerKey.Key, getExpiration(role)); err != nil {
			return err
		}

		if err := repo.SetThreshold(role, 1); err != nil {
			return err
		}
	}

	// Add targets (copy them into the repository and add them to the targets.json)
	expectedTargets := make(map[string]bool)
	for tt, custom := range targetsConfig {
		from, err := os.Open(tt)
		if err != nil {
			return err
		}
		defer from.Close()
		base := filepath.Base(tt)
		to, err := os.Create(filepath.Join(directory, "staged", "targets", base))
		if err != nil {
			return err
		}
		defer to.Close()
		if _, err := io.Copy(to, from); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Created target file at ", to.Name())
		if err := repo.AddTargetWithExpiresToPreferredRole(base, custom, getExpiration("targets"), "targets"); err != nil {
			return fmt.Errorf("error adding targets %w", err)
		}
		expectedTargets[base] = true
	}

	// Remove old targets that were not included in config.
	targetsToRemove := []string{}
	allTargets, err := repo.Targets()
	if err != nil {
		return err
	}
	for path := range allTargets {
		if !expectedTargets[path] {
			targetsToRemove = append(targetsToRemove, path)
		}
	}
	// Only call RemoveTargetsWithExpires if there are targets to remove.
	// Calling the function with an empty slice will remove all targets.
	if len(targetsToRemove) > 0 {
		if err := repo.RemoveTargetsWithExpires(targetsToRemove, getExpiration("targets")); err != nil {
			return fmt.Errorf("error removing old targets: %w", err)
		}
	}

	if err := repo.SetThreshold("targets", threshold); err != nil {
		return err
	}

	// Add blank signatures to root and targets
	t, err := prepo.GetTargetsFromStore(store)
	if err != nil {
		return err
	}
	if err := setMetaWithSigKeyIDs(store, "targets.json", t, keys); err != nil {
		return err
	}

	// We manually increment the root version in case adding the verification keys did not
	// increment the root version (because of no change).
	root, err = prepo.GetRootFromStore(store)
	if err != nil {
		return err
	}
	root.Version = curRootVersion + 1
	root.Expires = getExpiration("root")
	root.ConsistentSnapshot = ConsistentSnapshot
	// TODO: Add a test that makes sure that root has allRootKeys
	return setMetaWithSigKeyIDs(store, "root.json", root, allRootKeys)
}

func setSignedMeta(store tuf.LocalStore, role string, s *data.Signed) error {
	b, err := jsonMarshal(s)
	if err != nil {
		return err
	}
	return store.SetMeta(role, b)
}

func setMetaWithSigKeyIDs(store tuf.LocalStore, role string, meta interface{}, keys []*data.PublicKey) error {
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

func ClearEmptySignatures(store tuf.LocalStore, role string) error {
	signedMeta, err := prepo.GetSignedMeta(store, role)
	if err != nil {
		return err
	}

	sigs := make([]data.Signature, 0, 1)
	for _, signature := range signedMeta.Signatures {
		if len(signature.Signature) == 0 {
			// Skip placeholder signatures.
			continue
		}
		sigs = append(sigs, signature)
	}

	return setSignedMeta(store, role, &data.Signed{Signatures: sigs, Signed: signedMeta.Signed})
}

func jsonMarshal(v interface{}) ([]byte, error) {
	// Note we no longer need to store canonicalized JSON in the store.
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "\t"); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// getKeysFromDir will return the TUF public keys for each key in the
// directory.
func getKeysFromDir(dir string) ([]*data.PublicKey, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var tufKeys []*data.PublicKey
	for _, file := range files {
		if file.IsDir() {
			key, err := pkeys.SigningKeyFromDir(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			// v5 note!
			// We are now returning PEM-encoded ECDSA keys from the directories.
			tufKey, err := pkeys.ToTufKey(*key, false)
			if err != nil {
				return nil, err
			}
			tufKeys = append(tufKeys, tufKey)

		}
	}
	return tufKeys, err
}
