//
// Copyright 2022 The Sigstore Authors.
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
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/pkg/keys"
)

func TestGetSigningKeyIDs(t *testing.T) {
	tests := []struct {
		name           string
		dir            string
		role           string
		expectedKeyIds []string
		shouldErr      bool
	}{
		{
			name: "root role - initial root",
			dir:  "./testdata/single_root",
			role: "root",
			expectedKeyIds: []string{
				"2f64fb5eac0cf94dd39bb45308b98920055e9a0d8e012a7220787834c60aef97",
				"bdde902f5ec668179ff5ca0dabf7657109287d690bf97e230c21d65f99155c62",
				"eaf22372f417dd618a46f6c627dbc276e9fd30a004fc94f9be946e73f8bd090b",
				"f40f32044071a9365505da3d1e3be6561f6f22d0e60cf51df783999f6c3429cb",
				"f505595165a177a41750a8e864ed1719b1edfccd5a426fd2c0ffda33ce7ff209",
			},
			shouldErr: false,
		},
		{
			name: "rotated root keys role",
			dir:  "./testdata/multiple_root",
			role: "root",
			expectedKeyIds: []string{
				"2f64fb5eac0cf94dd39bb45308b98920055e9a0d8e012a7220787834c60aef97",
				"bdde902f5ec668179ff5ca0dabf7657109287d690bf97e230c21d65f99155c62",
				"eaf22372f417dd618a46f6c627dbc276e9fd30a004fc94f9be946e73f8bd090b",
				"f40f32044071a9365505da3d1e3be6561f6f22d0e60cf51df783999f6c3429cb",
				"f505595165a177a41750a8e864ed1719b1edfccd5a426fd2c0ffda33ce7ff209",
				"75e867ab10e121fdef32094af634707f43ddd79c6bab8ad6c5ab9f03f4ea8c90",
			},
			shouldErr: false,
		},
		{
			name:      "missing role",
			dir:       "./testdata/single_root",
			role:      "foo",
			shouldErr: true,
		},
		{
			name: "delegated role",
			dir:  "./testdata/multiple_root",
			role: "revocation",
			expectedKeyIds: []string{
				"9e7d813e8e16062e60a4540346aa8e7c7782afb7098af0b944ea80a4033a176f",
			},
			shouldErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			dirPath := filepath.Join(wd, tt.dir)
			store := tuf.FileSystemStore(dirPath, nil)

			keys, err := GetSigningKeyIDsForRole(tt.role, store)
			if tt.shouldErr != (err != nil) {
				t.Fatalf("GetSigningKeyIDsForRole() error: expected shouldErr %t, got %s",
					tt.shouldErr, err)
			}
			if err != nil {
				return
			}

			for _, key := range tt.expectedKeyIds {
				if _, ok := keys[key]; !ok {
					t.Errorf("expected key %s in signing keys for role %s", key, tt.role)
				}
			}
			if len(keys) != len(tt.expectedKeyIds) {
				t.Errorf("expected %d signing keys, got %d", len(tt.expectedKeyIds), len(keys))
			}
		})
	}
}

func TestUpdateRoleKeys(t *testing.T) {
	td := t.TempDir()
	store := tuf.FileSystemStore(td, nil)
	repo, err := tuf.NewRepo(store)
	if err != nil {
		t.Fatal(err)
	}
	rootSigner, err := keys.GenerateEcdsaKey()
	if err != nil {
		t.Fatal(err)
	}
	pk := rootSigner.PublicData()
	// Check on first role update that the key is added.
	if err := UpdateRoleKeys(repo, store, "root",
		[]*data.PublicKey{pk}, data.DefaultExpires("root")); err != nil {
		t.Fatal(err)
	}
	rootKeys, err := repo.RootKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(rootKeys) != 1 {
		t.Errorf("expected 1 root key in the repo, got %d", len(rootKeys))
	}
	if !reflect.DeepEqual(rootKeys[0], pk) {
		t.Errorf("expected root key with ID %s, got %s", pk.IDs()[0], rootKeys[0].IDs()[0])
	}
	// Now update again, and expect the old was revoked.
	newRootSigner, err := keys.GenerateEcdsaKey()
	if err != nil {
		t.Fatal(err)
	}
	newPk := newRootSigner.PublicData()
	// Check on first role update that the key is added.
	if err := UpdateRoleKeys(repo, store, "root",
		[]*data.PublicKey{newPk}, data.DefaultExpires("root")); err != nil {
		t.Fatal(err)
	}
	rootKeys, err = repo.RootKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(rootKeys) != 1 {
		t.Errorf("expected 1 root key in the repo, got %d", len(rootKeys))
	}
	if !reflect.DeepEqual(rootKeys[0], newPk) {
		t.Errorf("expected root key with ID %s, got %s", newPk.IDs()[0], rootKeys[0].IDs()[0])
	}
	// Now update with both, and expect both.
	if err := UpdateRoleKeys(repo, store, "root",
		[]*data.PublicKey{newPk, pk}, data.DefaultExpires("root")); err != nil {
		t.Fatal(err)
	}
	rootKeys, err = repo.RootKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(rootKeys) != 2 {
		t.Errorf("expected 2 root key in the repo, got %d", len(rootKeys))
	}
	if !reflect.DeepEqual(rootKeys, []*data.PublicKey{newPk, pk}) {
		t.Errorf("expected root keys with IDs %s %s", newPk.IDs()[0], pk.IDs()[0])
	}
}

func TestGetPreviousRootFromStore(t *testing.T) {
	tests := []struct {
		name            string
		dir             string
		expectedVersion int
		shouldErr       bool
	}{
		{
			name:      "root role - initial root",
			dir:       "./testdata/single_root",
			shouldErr: true,
		},
		{
			name:            "rotated root keys role",
			dir:             "./testdata/multiple_root",
			expectedVersion: 3,
			shouldErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			dirPath := filepath.Join(wd, tt.dir)
			store := tuf.FileSystemStore(dirPath, nil)
			prevRoot, err := GetPreviousRootFromStore(store)
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error %t got %s", tt.shouldErr, err)
			}
			if err != nil {
				return
			}
			if prevRoot.Version != int64(tt.expectedVersion) {
				t.Errorf("expected prev root version %d, got %d", tt.expectedVersion, prevRoot.Version)
			}
		})
	}
}
