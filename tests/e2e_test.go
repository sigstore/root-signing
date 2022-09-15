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

//go:build pivkey
// +build pivkey

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/secure-systems-lab/go-securesystemslib/cjson"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/root-signing/cmd/tuf/app"
	vapp "github.com/sigstore/root-signing/cmd/verify/app"
	"github.com/sigstore/root-signing/pkg/keys"
	prepo "github.com/sigstore/root-signing/pkg/repo"
	stuf "github.com/sigstore/sigstore/pkg/tuf"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
	tufkeys "github.com/theupdateframework/go-tuf/pkg/keys"
)

// TODO(asraa): Add more unit tests, including
//   * Custom metadata included in targets

// Create a test HSM key located in a keys/ subdirectory of testDir.
func createTestHsmKey(testDir string) error {
	keyDir := filepath.Join(testDir, "keys", "10550341")
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return err
	}

	testKey := "test_data/10550341"
	wd, _ := os.Getwd()
	testKey = filepath.Join(wd, testKey)

	return filepath.Walk(testKey, func(path string, info os.FileInfo, err error) error {
		var relPath string = strings.Replace(path, testKey, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(keyDir, relPath), 0755)
		} else {
			var data, err1 = ioutil.ReadFile(filepath.Join(testKey, relPath))
			if err1 != nil {
				return err1
			}
			return ioutil.WriteFile(filepath.Join(keyDir, relPath), data, 0777)
		}
	})
}

// Create fake key signer in testDirectory. Returns file reference to signer.
func createTestSigner(t *testing.T) string {
	keys, err := cosign.GenerateKeyPair(nil)
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	f, _ := os.CreateTemp(temp, "*.key")

	if _, err := io.Copy(f, bytes.NewBuffer(keys.PrivateBytes)); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

// Verify with the go-tuf client, sigstore-tuf, and our CLI verification.
// Note! Sigstore TUF uses a singleton to cache network calls. Disable this
// because if state changes during the test, Sigstore TUF won't update.
// TODO: https://github.com/sigstore/sigstore/issues/686
func verifyTuf(t *testing.T, repo string, root []byte) (data.TargetFiles, error) {
	td := t.TempDir()
	remote, err := vapp.FileRemoteStore(repo)
	if err != nil {
		t.Fatal(err)
	}
	local := client.MemoryLocalStore()
	c := client.NewClient(local, remote)
	if err := c.Init(root); err != nil {
		t.Fatal(err)

	}
	targets, err := c.Update()
	if err != nil {
		t.Fatal(err)
	}

	// Verify with root-signing verify CLI
	rootFile := filepath.Join(td, "root.json")
	if err := os.WriteFile(rootFile, root, 0600); err != nil {
		t.Fatal(err)
	}
	if err := vapp.VerifyCmd(false, repo, rootFile, nil, nil); err != nil {
		t.Fatal(err)
	}

	return targets, nil
}

// checkMetadataVersion checks that the roles are at version.
func checkMetadataVersion(t *testing.T, repo string, roles []string, version int) {
	store := tuf.FileSystemStore(repo, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	for _, metaName := range roles {
		md, ok := meta[metaName]
		if !ok {
			t.Fatalf("missing %s", metaName)
		}
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		sm, err := vapp.PrintAndGetSignedMeta(metaName, signed.Signed)
		if err != nil {
			t.Fatal(err)
		}
		if sm.Version != version {
			t.Errorf("expected %s version 2, got %d", metaName, sm.Version)
		}
	}
}

func verifySigstoreTuf(t *testing.T, repo string, root []byte,
	checkSigstoreTargets bool) error {
	// Verify with sigstore TUF
	td := t.TempDir()
	t.Setenv(stuf.TufRootEnv, td)
	ctx := context.Background()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir(filepath.Join(repo, "repository"))),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	}()
	if err := stuf.Initialize(ctx, "http://localhost:8080", root); err != nil {
		t.Fatal(err)
	}
	status, err := stuf.GetRootStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tufObj, err := stuf.NewFromEnv(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range status.Targets {
		if _, err := tufObj.GetTarget(target); err != nil {
			t.Fatalf("expected target %s, targets are: %s", target,
				strings.Join(status.Targets, ", "))
		}
	}
	return nil
}

func snapshotTimestampPublish(ctx context.Context, t *testing.T, repo string,
	snapshotKey, timestampKey string) {
	if err := app.SnapshotCmd(ctx, repo); err != nil {
		t.Fatalf("expected Snapshot command to pass, got err: %s", err)
	}
	snapshotSigner, err := keys.GetSigningKey(ctx, snapshotKey, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, repo, []string{"snapshot"}, snapshotSigner); err != nil {
		t.Fatal(err)
	}

	if err := app.TimestampCmd(ctx, repo); err != nil {
		t.Fatalf("expected Timestamp command to pass, got err: %s", err)
	}
	timestampSigner, err := keys.GetSigningKey(ctx, timestampKey, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, repo, []string{"timestamp"}, timestampSigner); err != nil {
		t.Fatal(err)
	}

	// Successful Publishing!
	if err := app.PublishCmd(ctx, repo); err != nil {
		t.Fatal(err)
	}
}

func TestInitCmd(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := createTestHsmKey(td); err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Verify that root and targets have expected version 1 on Init.
	checkMetadataVersion(t, td,
		[]string{"root.json", "targets.json"},
		1)
}

func TestSignRootTargets(t *testing.T) {
	// Initialize.

	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	serial, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize with 1 succeeds.
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root and targets.
	signerAndKey, err := GetTestHsmSigner(ctx, td, *serial, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, signerAndKey); err != nil {
		t.Fatal(err)
	}

	// Verify that root and targets have one signature.
	store := tuf.FileSystemStore(td, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	for _, metaName := range []string{"root.json", "targets.json"} {
		md, ok := meta[metaName]
		if !ok {
			t.Fatalf("missing %s", metaName)
		}
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		if len(signed.Signatures) != 1 {
			t.Fatalf("missing signatures on %s", metaName)
		}
		if !signerAndKey.Key.ContainsID(signed.Signatures[0].KeyID) {
			t.Fatalf("missing key id for signer on %s", metaName)
		}
		if len(signed.Signatures[0].Signature) == 0 {
			t.Fatalf("missing signature on %s", metaName)
		}
	}
}

func TestSnapshotUnvalidatedFails(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	rootkey1, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Validate that root and targets have one unfilled signature.
	store := tuf.FileSystemStore(td, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	for _, metaName := range []string{"root.json", "targets.json"} {
		md, ok := meta[metaName]
		if !ok {
			t.Fatalf("missing %s", metaName)
		}
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		if len(signed.Signatures) != 2 {
			t.Fatalf("expected 1 signature on %s", metaName)
		}
		if len(signed.Signatures[0].Signature) != 0 {
			t.Fatalf("expected empty signature for key ID %s", signed.Signatures[0].KeyID)
		}
	}

	// Try to snapshot. Expect to fail.
	if err := app.SnapshotCmd(ctx, td); err == nil {
		t.Fatalf("expected Snapshot command to fail")
	}

	// Now sign root and targets with 1/1 threshold key.
	signerAndKey1, err := GetTestHsmSigner(ctx, td, *rootkey1, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, signerAndKey1); err != nil {
		t.Fatal(err)
	}

	// Expect that there is still one empty placeholder signature.
	store = tuf.FileSystemStore(td, nil)
	meta, err = store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	for _, metaName := range []string{"root.json", "targets.json"} {
		md, ok := meta[metaName]
		if !ok {
			t.Fatalf("missing %s", metaName)
		}
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		if len(signed.Signatures) != 2 {
			t.Fatalf("expected 2 signature on %s, got %d", metaName, len(signed.Signatures))
		}
		if len(signed.Signatures[0].Signature) != 0 && len(signed.Signatures[1].Signature) != 0 {
			t.Fatalf("expected one empty signature")
		}
	}

	// Snapshot success! We clear the empty placeholder signature in root/targets.
	if err := app.SnapshotCmd(ctx, td); err != nil {
		t.Fatalf("expected Snapshot command to pass, got err: %s", err)
	}
}

func TestPublishSuccess(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	rootSerial, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootKey, err := GetTestHsmSigner(ctx, td, *rootSerial, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Check versions.
	store := tuf.FileSystemStore(td, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	for _, metaName := range []string{"root.json", "targets.json", "snapshot.json", "timestamp.json"} {
		md, ok := meta[metaName]
		if !ok {
			t.Fatalf("missing %s", metaName)
		}
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		sm, err := vapp.PrintAndGetSignedMeta(metaName, signed.Signed)
		if err != nil {
			t.Fatal(err)
		}
		if sm.Version != 1 {
			t.Errorf("expected metadata version 1, got %d", sm.Version)
		}
	}

	// Verify with go-tuf
	targetFiles, err := verifyTuf(t, td, meta["root.json"])
	if err != nil {
		t.Fatal(err)
	}
	if len(targetFiles) != 1 {
		t.Fatalf("expected one target, got %d", len(targetFiles))
	}
	for name := range targetFiles {
		if !strings.EqualFold(name, "foo.txt") {
			t.Fatalf("expected one target foo.txt, got %s", name)
		}
	}
}

func TestRotateRootKey(t *testing.T) {
	// This tests root key rotation: we use a threshold of 1 with 2 root keys
	// and expect to rotate one keyholder during an update.
	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	rootSerial1, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}
	rootSerial2, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets with key 1
	rootKey1, err := GetTestHsmSigner(ctx, td, *rootSerial1, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey1); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Check that there are two root key signers: key 1 and key 2.
	store := tuf.FileSystemStore(td, nil)
	root, err := prepo.GetRootFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	rootRole, ok := root.Roles["root"]
	if !ok {
		t.Fatalf("expected root role")
	}
	rootKey2, err := GetTestHsmSigner(ctx, td, *rootSerial2, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	expectedKeyIds := append(rootKey1.Key.IDs(), rootKey2.Key.IDs()...)
	actualKeyIds := rootRole.KeyIDs
	sort.Strings(expectedKeyIds)
	sort.Strings(actualKeyIds)
	if !cmp.Equal(expectedKeyIds, actualKeyIds) {
		t.Fatalf("expected key IDs %s, got %s", expectedKeyIds, actualKeyIds)
	}

	// Now remove the second key and add a third key.
	if err := os.RemoveAll(filepath.Join(td, "keys", fmt.Sprint(*rootSerial2))); err != nil {
		t.Fatal(err)
	}
	rootSerial3, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new root.
	if err := app.InitCmd(ctx, td, td, 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Check the root keys were rotated: expect key 1 and 3.
	root, err = prepo.GetRootFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	rootRole, ok = root.Roles["root"]
	if !ok {
		t.Fatalf("expected root role")
	}
	rootKey3, err := GetTestHsmSigner(ctx, td, *rootSerial3, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	expectedKeyIds = append(rootKey1.Key.IDs(), rootKey3.Key.IDs()...)
	actualKeyIds = rootRole.KeyIDs
	sort.Strings(expectedKeyIds)
	sort.Strings(actualKeyIds)
	if !cmp.Equal(expectedKeyIds, actualKeyIds) {
		t.Fatalf("expected key IDs %s, got %s", expectedKeyIds, actualKeyIds)
	}

	// Expect version 2 for root.
	if root.Version != 2 {
		t.Fatalf("expected root version 2, got %d", root.Version)
	}

	// Sign root & targets
	rootKey, err := GetTestHsmSigner(ctx, td, *rootSerial1, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Verify with go-tuf
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = verifyTuf(t, td, meta["root.json"]); err != nil {
		t.Fatal(err)
	}
}

func TestRotateTarget(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	rootSerial, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootKey, err := GetTestHsmSigner(ctx, td, *rootSerial, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Check versions.
	checkMetadataVersion(t, td,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		1)

	// Verify with go-tuf
	store := tuf.FileSystemStore(td, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}

	// Verify with go-tuf
	targetFiles, err := verifyTuf(t, td, meta["root.json"])
	if err != nil {
		t.Fatal(err)
	}
	if len(targetFiles) != 1 {
		t.Fatalf("expected one target, got %d", len(targetFiles))
	}
	for name := range targetFiles {
		if !strings.EqualFold(name, "foo.txt") {
			t.Fatalf("expected one target foo.txt, got %s", name)
		}
	}

	// New target, config only targets new file, not old
	testTarget = filepath.Join(td, "bar.txt")
	targetsConfig = map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abcdef"), 0600); err != nil {
		t.Fatal(err)
	}

	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, td, 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Check versions.
	checkMetadataVersion(t, td,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		2)

	// Verify with go-tuf
	meta, err = store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}

	// Verify with go-tuf
	targetFiles, err = verifyTuf(t, td, meta["root.json"])
	if err != nil {
		t.Fatal(err)
	}
	if len(targetFiles) != 1 {
		t.Fatalf("expected one target, got %d", len(targetFiles))
	}
	for name := range targetFiles {
		if !strings.EqualFold(name, "bar.txt") {
			t.Fatalf("expected one target bar.txt, got %s", name)
		}
	}
}

// Tests that enabling consistent snapshots at version > 1 works.
func TestConsistentSnapshotFlip(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	rootSerial, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds with consistent snapshot off.
	app.ConsistentSnapshot = false
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootKey, err := GetTestHsmSigner(ctx, td, *rootSerial, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Check versions.
	checkMetadataVersion(t, td,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		1)

	// Verify with go-tuf
	store := tuf.FileSystemStore(td, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	targetFiles, err := verifyTuf(t, td, meta["root.json"])
	if err != nil {
		t.Fatal(err)
	}
	if len(targetFiles) != 1 {
		t.Fatalf("expected one target, got %d", len(targetFiles))
	}
	for name := range targetFiles {
		if !strings.EqualFold(name, "foo.txt") {
			t.Fatalf("expected one target foo.txt, got %s", name)
		}
	}

	// Flip consistent snapshot on.
	app.ConsistentSnapshot = true
	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, td, 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Check versions.
	checkMetadataVersion(t, td,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		2)

	// Verify with TUF
	targetFiles, err = verifyTuf(t, td, meta["root.json"])
	if err != nil {
		t.Fatal(err)
	}
	if len(targetFiles) != 1 {
		t.Fatalf("expected one target, got %d", len(targetFiles))
	}
	for name := range targetFiles {
		if !strings.EqualFold(name, "foo.txt") {
			t.Fatalf("expected one target foo.txt, got %s", name)
		}
	}

	if _, err = verifyTuf(t, td, meta["root.json"]); err != nil {
		t.Fatal(err)
	}

	// Verify consistent snapshotting was enabled by
	// checking that 2.snapshot.json is present.
	repoFiles, err := ioutil.ReadDir(filepath.Join(td, "repository"))
	if err != nil {
		t.Fatal(err)
	}
	foundSnapshot := false
	for _, file := range repoFiles {
		if file.Name() == "2.snapshot.json" {
			foundSnapshot = true
			break
		}
	}
	if !foundSnapshot {
		t.Fatal("expected 2.snapshot.json in consistent snapshotted repo")
	}
}

func TestSignWithEcdsaHexHSM(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	rootSerial, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, app.DeprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Sign root
	rootKey, err := GetTestHsmSigner(ctx, td, *rootSerial, app.DeprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Verify that root has one signature using hex-encoded key.
	store := tuf.FileSystemStore(td, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}

	md, ok := meta["root.json"]
	if !ok {
		t.Fatalf("missing root")
	}
	signed := &data.Signed{}
	if err := json.Unmarshal(md, signed); err != nil {
		t.Fatal(err)
	}
	if len(signed.Signatures) != 1 {
		t.Fatalf("missing signatures on root")
	}
	if !rootKey.Key.ContainsID(signed.Signatures[0].KeyID) {
		t.Fatalf("missing key id for signer on root")
	}
	if len(signed.Signatures[0].Signature) == 0 {
		t.Fatalf("missing signature on root")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(signed.Signed, &decoded); err != nil {
		t.Fatal(err)
	}
	msg, err := cjson.EncodeCanonical(decoded)
	if err != nil {
		t.Fatal(err)
	}

	// Use the deprecated ECDSA verifier from TUF that uses hex-encoded keys.
	deprecatedVerifier := tufkeys.NewDeprecatedEcdsaVerifier()
	if err := deprecatedVerifier.UnmarshalPublicKey(rootKey.Key); err != nil {
		t.Fatalf("error unmarshalling deprecated hex key")
	}
	if err := deprecatedVerifier.Verify(msg, signed.Signatures[0].Signature); err != nil {
		t.Fatalf("error verifying signature")
	}
}

func TestEcdsaHexToPEMMigration(t *testing.T) {
	ctx := context.Background()
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	testTarget := filepath.Join(td, "foo.txt")
	targetsConfig := map[string]json.RawMessage{testTarget: nil}

	if err := os.WriteFile(testTarget, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}

	rootSerial, err := CreateTestHsmSigner(td, rootCA, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	snapshotKey := createTestSigner(t)
	timestampKey := createTestSigner(t)

	// Initialize succeeds with deprecated format.
	deprecatedEcdsaFormat := true
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey, deprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Just to make sure, try this with the un-deprecated signer and expect failure
	rootKeyPEM, err := GetTestHsmSigner(ctx, td, *rootSerial, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root"}, rootKeyPEM); err == nil {
		t.Fatal("expected error signing with PEM key")
	}

	// Sign root with deprecated format signer.
	rootKeyHex, err := GetTestHsmSigner(ctx, td, *rootSerial, deprecatedEcdsaFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKeyHex); err != nil {
		t.Fatal(err)
	}

	// Finish publishing & verify
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)
	store := tuf.FileSystemStore(td, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	initialSignedRoot := meta["root.json"]
	verifyTuf(t, td, initialSignedRoot)
	checkMetadataVersion(t, td,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		1)

	// Flip the format and re-init! This should add "new" TUF key, same material.
	deprecatedEcdsaFormat = false
	if err := app.InitCmd(ctx, td, td, 1, targetsConfig, snapshotKey, timestampKey, deprecatedEcdsaFormat); err != nil {
		t.Fatal(err)
	}

	// Check that only the PEM key IDs are on root & targets role.
	meta, err = store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	root, err := prepo.GetRootFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, roleName := range []string{"root", "targets"} {
		role, ok := root.Roles["root"]
		if !ok {
			t.Fatalf("expected %s role", roleName)
		}
		if len(role.KeyIDs) != 1 {
			t.Fatal("expected 1 key IDs on root role")
		}
		if role.KeyIDs[0] != rootKeyPEM.Key.IDs()[0] {
			t.Fatal("expected PEM ECDSA TUF key")
		}
	}

	// Check that there are 2 signature pre-entries on root
	// one for Hex and one for PEM.
	md, ok := meta["root.json"]
	if !ok {
		t.Fatalf("missing root")
	}
	signed := &data.Signed{}
	if err := json.Unmarshal(md, signed); err != nil {
		t.Fatal(err)
	}
	if len(signed.Signatures) != 2 {
		t.Fatalf("expected 2 signature pre-entries on root, got %d", len(signed.Signatures))
	}
	var preEntries []string
	for _, sig := range signed.Signatures {
		preEntries = append(preEntries, sig.KeyID)
	}
	sort.Strings(preEntries)
	expectedKeyIds := append(rootKeyHex.Key.IDs(), rootKeyPEM.Key.IDs()...)
	sort.Strings(expectedKeyIds)
	if !cmp.Equal(expectedKeyIds, preEntries) {
		t.Fatalf("expected key IDs %s, got %s", expectedKeyIds, preEntries)
	}

	// Check that there is 1 signature pre-entry on targets, just the new PEM.
	md, ok = meta["targets.json"]
	if !ok {
		t.Fatalf("missing targets")
	}
	signed = &data.Signed{}
	if err := json.Unmarshal(md, signed); err != nil {
		t.Fatal(err)
	}
	if len(signed.Signatures) != 1 {
		t.Fatalf("expected 1 signature pre-entries on targets, got %d", len(signed.Signatures))
	}
	if !cmp.Equal(signed.Signatures[0].KeyID, rootKeyPEM.Key.IDs()[0]) {
		t.Fatalf("expected key IDs %s, got %s", expectedKeyIds, preEntries)
	}

	// Now sign with both key types.
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKeyPEM); err != nil {
		t.Fatal(err)
	}

	// Sign root with deprecated format signer just on the root.
	if err := app.SignCmd(ctx, td, []string{"root"}, rootKeyHex); err != nil {
		t.Fatal(err)
	}

	// Finish publishing.
	snapshotTimestampPublish(ctx, t, td, snapshotKey, timestampKey)

	// Check version 2.
	checkMetadataVersion(t, td,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		2)
	// Verify using the bytes of 1.root.json.
	verifyTuf(t, td, initialSignedRoot)
	// Verify using the bytes of 2.root.json.
	meta, err = store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	verifyTuf(t, td, meta["root.json"])
}
