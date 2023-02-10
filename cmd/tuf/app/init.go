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
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	csignature "github.com/sigstore/cosign/pkg/signature"
	pkeys "github.com/sigstore/root-signing/pkg/keys"
	prepo "github.com/sigstore/root-signing/pkg/repo"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"golang.org/x/exp/maps"
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

func GetExpiration(role string) time.Time {
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
		flagset     = flag.NewFlagSet("tuf init", flag.ExitOnError)
		repository  = flagset.String("repository", "", "path to initialize the staged repository")
		threshold   = flagset.Int("threshold", DefaultThreshold, "default root and targets signer threshold")
		previous    = flagset.String("previous", "", "path to the previous repository")
		snapshot    = flagset.String("snapshot", "", "reference to an online snapshot signer")
		timestamp   = flagset.String("timestamp", "", "reference to an online timestamp signer")
		targetsMeta = flagset.String("target-meta", "", "path to a target configuration file")
		targetsDir  = flagset.String("targets", "", "path to a target configuration file")
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
			if *targetsMeta == "" {
				return flag.ErrHelp
			}
			if *targetsDir == "" {
				return flag.ErrHelp
			}
			targetsConfigStr, err := os.ReadFile(*targetsMeta)
			if err != nil {
				return err
			}
			// targets config path is relative to wd
			targetsConfig, err := prepo.SigstoreTargetMetaFromString(targetsConfigStr)
			if err != nil {
				return err
			}
			return InitCmd(ctx, *repository, *previous,
				*threshold, targetsConfig,
				*targetsDir,
				*snapshot, *timestamp)
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
//   - targetsDir: The local directory where the targets are stored.
//   - snapshotRef: A reference (KMS, file, URL) to a snapshot signer.
//   - timestampRef: A reference (KMS, file, URL) to a timestamp signer.
//
// The root and targets metadata will be initialized with a 6 month expiration.
// Revoked keys will be automatically calculated given the previous root and the signers in directory.
// Signature placeholders for each key will be added to the root.json and targets.json file.
func InitCmd(ctx context.Context, directory, previous string,
	threshold int, targetsConfig *prepo.TargetMetaConfig,
	targetsDir string,
	snapshotRef string, timestampRef string) error {
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
	keys, err := getKeysFromDir(directory + "/keys")
	if err != nil {
		return fmt.Errorf("getting HSM keys: %s", err)
	}
	// Add any keys in the keys/ subfolder to root and targets.
	for _, role := range []string{"root", "targets"} {
		if err := prepo.UpdateRoleKeys(repo, store, role, keys,
			GetExpiration(role)); err != nil {
			return err
		}
		if err := repo.SetThreshold(role, threshold); err != nil {
			return err
		}
	}

	// Add keys used for snapshot and timestamp roles.
	for role, keyRef := range map[string]string{"snapshot": snapshotRef, "timestamp": timestampRef} {
		signer, err := csignature.SignerVerifierFromKeyRef(ctx, keyRef, nil)
		if err != nil {
			return err
		}

		// Construct TUF key.
		tufKey, err := pkeys.ConstructTufKey(ctx, signer)
		if err != nil {
			return err
		}

		// Update keys.
		if err := prepo.UpdateRoleKeys(repo, store, role, []*data.PublicKey{tufKey},
			GetExpiration(role)); err != nil {
			return err
		}

		// Set threshold.
		if err := repo.SetThreshold(role, 1); err != nil {
			return err
		}
	}

	// Add targets (copy them into the repository and add them to the targets.json)
	// Add the new targets in the config.
	expectedTargets := make(map[string]bool)
	for tt, custom := range targetsConfig.Add {
		from, err := os.Open(filepath.Join(targetsDir, tt))
		if err != nil {
			return err
		}
		defer from.Close()
		stagedPath := filepath.Join(directory, "staged", "targets", tt)
		if err := os.MkdirAll(filepath.Dir(stagedPath), 0770); err != nil {
			return err
		}
		to, err := os.Create(stagedPath)
		if err != nil {
			return err
		}
		defer to.Close()
		if _, err := io.Copy(to, from); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Created target file at ", to.Name())
		if err := repo.AddTargetWithExpiresToPreferredRole(tt, custom, GetExpiration("targets"), "targets"); err != nil {
			return fmt.Errorf("error adding targets %w", err)
		}
		expectedTargets[tt] = true
	}
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

	if err := repo.SetThreshold("targets", threshold); err != nil {
		return err
	}

	// Add blank signatures to root and targets
	t, err := prepo.GetTargetsFromStore(store)
	if err != nil {
		return err
	}
	// We reset and delegations: they will be updated in DelegationCmd.
	// TODO(https://github.com/theupdateframework/go-tuf/issues/402): replace
	// with `repo.ResetTargetsDelegationsWithExpires` when this issue is resolved.
	t.Delegations = nil
	targetKeys, err := prepo.GetSigningKeyIDsForRole("targets", store)
	if err != nil {
		return err
	}
	// Remove any targets not present: only removes from targets to avoid
	// removing from delegations.
	// See https://github.com/theupdateframework/go-tuf/issues/400 and
	// https://github.com/theupdateframework/go-tuf/blob/f75cbcc8550dfb9311c6723999fe7b1d3d2bc116/repo.go#L1230
	// for why we avoid `repo.RemoveTargetsWithExpires`
	for _, tt := range targetsToRemove {
		delete(t.Targets, tt)
	}

	if err := setMetaWithSigKeyIDs(store, "targets.json", t, maps.Keys(targetKeys)); err != nil {
		return err
	}

	// We manually increment the root version in case adding the verification keys did not
	// increment the root version (because of no change).
	root, err := prepo.GetRootFromStore(store)
	if err != nil {
		return err
	}
	root.Version = curRootVersion + 1
	root.Expires = GetExpiration("root")
	root.ConsistentSnapshot = ConsistentSnapshot
	allRootKeys, err := prepo.GetSigningKeyIDsForRole("root", store)
	if err != nil {
		return err
	}
	return setMetaWithSigKeyIDs(store, "root.json", root, maps.Keys(allRootKeys))
}

func setMetaWithSigKeyIDs(store tuf.LocalStore, role string, meta interface{}, keyIDs []string) error {
	signed, err := prepo.MarshalMetadata(meta)
	if err != nil {
		return err
	}

	// Add empty sigs
	emptySigs := make([]data.Signature, 0, 1)

	for _, id := range keyIDs {
		emptySigs = append(emptySigs, data.Signature{
			KeyID: id,
		})

	}

	return prepo.SetSignedMeta(store, role, &data.Signed{Signatures: emptySigs, Signed: signed})
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

	return prepo.SetSignedMeta(store, role, &data.Signed{Signatures: sigs, Signed: signedMeta.Signed})
}

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
			tufKey, err := pkeys.EcdsaTufKey(key.PublicKey)
			if err != nil {
				return nil, err
			}
			tufKeys = append(tufKeys, tufKey)
		}
	}
	return tufKeys, err
}
