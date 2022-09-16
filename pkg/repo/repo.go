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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ctuf "github.com/sigstore/sigstore/pkg/tuf"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/verify"
	"gopkg.in/yaml.v2"
)

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

func GetVersionedRootFromStore(store tuf.LocalStore, version int) (*data.Root, error) {
	s, err := GetSignedMeta(store, fmt.Sprintf("%d.root.json", version))
	if err != nil {
		return nil, err
	}
	root := &data.Root{}
	if err := json.Unmarshal(s.Signed, root); err != nil {
		return nil, err
	}
	return root, nil
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

type customMetadata struct {
	Usage  ctuf.UsageKind  `json:"usage"`
	Status ctuf.StatusKind `json:"status"`
}

type sigstoreCustomMetadata struct {
	Sigstore customMetadata `json:"sigstore"`
}

// Target metadata helpers
func SigstoreTargetMetaFromString(b []byte) (map[string]json.RawMessage, error) {
	targetsMeta := map[string]sigstoreCustomMetadata{}
	targetsMetaJSON := map[string]json.RawMessage{}
	if err := yaml.Unmarshal(b, &targetsMeta); err != nil {
		return nil, err
	}
	for t, custom := range targetsMeta {
		customJson, err := json.Marshal(custom)
		if err != nil {
			return nil, err
		}
		targetsMetaJSON[t] = customJson
	}

	return targetsMetaJSON, nil
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

// GetSigningKeyIDsForRole gets a map of signing key IDs for the given role.
// When this is a root role, checks if a previous root version exists
// and adds those: these keys should sign the role.
func GetSigningKeyIDsForRole(manifest string, store tuf.LocalStore) (map[string]bool, error) {
	res := make(map[string]bool, 0)
	name := strings.TrimSuffix(manifest, ".json")
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
			// Return the root role keys.
			return res, nil
		}
		// If this is a root role, check if there is a previous root.
		previousRoot, err := GetVersionedRootFromStore(store, int(root.Version-1))
		if err != nil {
			// No previous root.
			return res, nil
		}
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
