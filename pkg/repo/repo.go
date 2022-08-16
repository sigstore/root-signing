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

	ctuf "github.com/sigstore/cosign/pkg/cosign/tuf"
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
			// ignore ErrWrongID errors by TAP-12
			if _, ok := err.(verify.ErrWrongID); !ok {
				return nil, nil, err
			}
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
				// ignore ErrWrongID errors by TAP-12
				if _, ok := err.(verify.ErrWrongID); !ok {
					return nil, nil, err
				}
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
