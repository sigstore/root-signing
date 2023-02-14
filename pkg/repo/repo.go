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

package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/verify"
	kyaml "sigs.k8s.io/yaml"
)

var ErrNoPreviousRoot = errors.New("no previous root")

func CreateDb(store tuf.LocalStore) (db *verify.DB, thresholds map[string]int, err error) {
	db = verify.NewDB()
	root, err := GetRootFromStore(store)
	if err != nil {
		return nil, nil, err
	}
	for id, k := range root.Keys {
		if err := db.AddKey(id, k); err != nil {
			return nil, nil, err
		}
	}
	roles := make(map[string]bool, len(root.Roles))
	thresholds = make(map[string]int, len(root.Roles))
	for name, role := range root.Roles {
		if err := db.AddRole(name, role); err != nil {
			return nil, nil, err
		}
		thresholds[name] = role.Threshold
		roles[name] = true
	}

	// now add delegations
	meta, err := store.GetMeta()
	if err != nil {
		return nil, nil, err
	}
	for metaName, metadata := range meta {
		name := strings.TrimSuffix(metaName, ".json")
		if _, ok := roles[name]; ok && !strings.EqualFold(name, "targets") {
			continue
		}
		s := &data.Signed{}
		if err := json.Unmarshal(metadata, s); err != nil {
			return nil, nil, fmt.Errorf("error unmarshalling for targets %q: %w", metaName, err)
		}
		t := &data.Targets{}
		if err := json.Unmarshal(s.Signed, t); err != nil {
			return nil, nil, fmt.Errorf("error unmarshalling signed data for targets %q: %w", metaName, err)
		}

		if t.Delegations == nil {
			continue
		}

		for id, k := range t.Delegations.Keys {
			if err := db.AddKey(id, k); err != nil {
				return nil, nil, err
			}
		}
		for _, drole := range t.Delegations.Roles {
			role := &data.Role{Threshold: drole.Threshold, KeyIDs: drole.KeyIDs}
			thresholds[drole.Name] = drole.Threshold
			if err := db.AddRole(drole.Name, role); err != nil {
				return nil, nil, err
			}
		}
	}

	return db, thresholds, nil
}

func GetRootFromStore(store tuf.LocalStore) (*data.Root, error) {
	s, err := GetSignedMeta(store, "root.json")
	if err != nil {
		return nil, err
	}
	root := &data.Root{}
	if err := json.Unmarshal(s.Signed, root); err != nil {
		return nil, err
	}
	return root, nil
}

func GetPreviousRootFromStore(store tuf.LocalStore) (*data.Root, error) {
	root, err := GetRootFromStore(store)
	if err != nil {
		return nil, err
	}
	if root.Version < 2 {
		// No previous root.
		return nil, ErrNoPreviousRoot
	}
	s, err := GetSignedMeta(store, fmt.Sprintf("%d.root.json", int(root.Version-1)))
	if err != nil {
		return nil, err
	}
	prevRoot := &data.Root{}
	if err := json.Unmarshal(s.Signed, prevRoot); err != nil {
		return nil, err
	}
	return prevRoot, nil
}

func GetTargetsFromStore(store tuf.LocalStore) (*data.Targets, error) {
	s, err := GetSignedMeta(store, "targets.json")
	if err != nil {
		return nil, err
	}
	t := &data.Targets{}
	if err := json.Unmarshal(s.Signed, t); err != nil {
		return nil, err
	}
	return t, nil
}

func GetDelegatedTargetsFromStore(store tuf.LocalStore, manifest string) (*data.Targets, error) {
	s, err := GetSignedMeta(store, manifest)
	if err != nil {
		return nil, err
	}
	t := &data.Targets{}
	if err := json.Unmarshal(s.Signed, t); err != nil {
		return nil, err
	}
	return t, nil
}

func GetSignedMeta(store tuf.LocalStore, name string) (*data.Signed, error) {
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

func GetMetaFromStore(msg []byte, name string) (interface{}, error) {
	var meta interface{}
	switch name {
	case "root.json":
		meta = &data.Root{}
	case "snapshot.json":
		meta = &data.Snapshot{}
	case "timestamp.json":
		meta = &data.Timestamp{}
	default:
		// Targets and delegated roles
		meta = &data.Targets{}
	}
	if err := json.Unmarshal(msg, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// Target metadata helpers
func SigstoreTargetMetaFromString222(b []byte) (map[string]json.RawMessage, error) {
	jsonBytes, err := kyaml.YAMLToJSON(b)
	if err != nil {
		return nil, err
	}
	var targetsMetaJSON map[string]json.RawMessage
	if err := json.Unmarshal(jsonBytes, &targetsMetaJSON); err != nil {
		return nil, err
	}
	return targetsMetaJSON, nil
}

type TargetMetaConfig struct {
	Add map[string]json.RawMessage `json:"add"`
	Del map[string]json.RawMessage `json:"delete"`
}

func SigstoreTargetMetaFromString(b []byte) (*TargetMetaConfig, error) {
	jsonBytes, err := kyaml.YAMLToJSON(b)
	if err != nil {
		return nil, err
	}
	var targetsMetaJSON TargetMetaConfig
	if err := json.Unmarshal(jsonBytes, &targetsMetaJSON); err != nil {
		return nil, err
	}
	return &targetsMetaJSON, nil
}

func IsVersionedManifest(name string) bool {
	parts := strings.Split(name, ".")
	// Versioned manifests have the form "x.role.json"
	if len(parts) < 3 {
		return false
	}

	_, err := strconv.Atoi(parts[0])
	return err == nil
}

// GetSigningKeyIDsForRole returns a map of key IDs for the given role
// which are valid for signing.
// When this is a root role, the possible valid signing keys are
// the previous root role keys (if defined) and the current root role keys.
func GetSigningKeyIDsForRole(name string, store tuf.LocalStore) (
	map[string]bool, error) {
	res := make(map[string]bool, 0)
	root, err := GetRootFromStore(store)
	if err != nil {
		return nil, err
	}
	role, ok := root.Roles[name]
	if ok {
		for _, id := range role.KeyIDs {
			res[id] = true
		}
		if name != "root" {
			// Non-root roles only require signatures associated with the current
			// role keys.
			return res, nil
		}
		// If this is a root role, check if there is a previous root.
		previousRoot, err := GetPreviousRootFromStore(store)
		if err != nil {
			// No previous root.
			if errors.Is(err, ErrNoPreviousRoot) {
				return res, nil
			}
			return nil, err
		}
		fmt.Fprintf(os.Stderr, "Adding previous root keys from root version %d\n", previousRoot.Version)
		previousRootRole, ok := previousRoot.Roles[name]
		if !ok {
			return nil, fmt.Errorf("missing role %s on previous root", err)
		}
		for _, id := range previousRootRole.KeyIDs {
			res[id] = true
		}
		return res, nil
	}
	// This is a delegation.
	meta, err := store.GetMeta()
	if err != nil {
		return nil, err
	}
	for metaName := range meta {
		t, err := GetDelegatedTargetsFromStore(store, metaName)
		if err != nil {
			// This may not be a targets file.
			continue
		}
		if t.Delegations == nil {
			continue
		}
		for _, role := range t.Delegations.Roles {
			if name == role.Name {
				for _, id := range role.KeyIDs {
					res[id] = true
				}
				return res, nil
			}
		}
	}
	return res, fmt.Errorf("role %s not found", name)
}

// UpdateRoleKeys updates the roles keys, and handles the logic for
// revoking any old keys.
func UpdateRoleKeys(repo *tuf.Repo, store tuf.LocalStore, role string, keys []*data.PublicKey,
	expiration time.Time) error {
	currentKeyMap := map[string]bool{}
	for _, tufKey := range keys {
		currentKeyMap[tufKey.IDs()[0]] = true
		if err := repo.AddVerificationKeyWithExpiration(role, tufKey, expiration); err != nil {
			return err
		}
	}
	// Revoke any old keys from previous versions that are not explicitly added.
	root, err := GetRootFromStore(store)
	if err != nil {
		return err
	}
	oldKeys, ok := root.Roles[role]
	// Note: if the role does not exist, then there was no previous version of the role.
	if ok {
		for _, oldKeyID := range oldKeys.KeyIDs {
			if _, ok := currentKeyMap[oldKeyID]; !ok {
				if err := repo.RevokeKey(role, oldKeyID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func MarshalMetadata(v interface{}) ([]byte, error) {
	// We don't need to canonically encode the payload in the store.
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

func SetSignedMeta(store tuf.LocalStore, role string, s *data.Signed) error {
	b, err := MarshalMetadata(s)
	if err != nil {
		return err
	}
	return store.SetMeta(role, b)
}

// BumpMetadataVersion increments the version of the manifest.
// Note: This does NOT increase expiration!
// This ONLY handles delegated targets roles! The repo.SetRootVersion,
// repo.SetTargetsVersion, repo.SetSnapshotVersion, or repo.SetTimestampVersion
// handle top-level metadata.
func BumpMetadataVersion(store tuf.LocalStore, name string) error {
	for _, topName := range []string{"root", "targets", "snapshot", "timestamp"} {
		if name == topName {
			return fmt.Errorf("unsupported metadata version bump %s", topName)
		}
	}
	manifest := fmt.Sprintf("%s.json", name)
	s, err := GetSignedMeta(store, manifest)
	if err != nil {
		return err
	}
	targets := &data.Targets{}
	if err := json.Unmarshal(s.Signed, targets); err != nil {
		return err
	}
	targets.Version++

	signed, err := MarshalMetadata(targets)
	if err != nil {
		return err
	}

	return SetSignedMeta(store, manifest, &data.Signed{Signed: signed})
}
