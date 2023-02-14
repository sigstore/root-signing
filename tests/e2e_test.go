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
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sigstore/root-signing/cmd/tuf/app"
	vapp "github.com/sigstore/root-signing/cmd/verify/app"
	"github.com/sigstore/root-signing/pkg/keys"
	prepo "github.com/sigstore/root-signing/pkg/repo"
	stuf "github.com/sigstore/sigstore/pkg/tuf"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
)

// TODO(asraa): Add more unit tests, including
//   * Custom metadata included in targets

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

func TestInitCmd(t *testing.T) {
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)

	stack.addTarget(t, "foo.txt", "abc", nil)
	stack.genKey(t, true)

	if _, err := CreateTestHsmSigner(stack.repoDir, stack.hsmRootCA, stack.hsmRootSigner); err != nil {
		t.Fatal(err)
	}

	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Verify that root and targets have expected version 1 on Init.
	checkMetadataVersion(t, stack.repoDir,
		[]string{"root.json", "targets.json"},
		1)
}

func TestSignRootTargets(t *testing.T) {
	// Initialize.
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)

	// Initialize with 1 succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root and targets.
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	pubKey, err := keys.ConstructTufKey(ctx, rootSigner)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that root and targets have one signature.
	for _, metaName := range []string{"root.json", "targets.json"} {
		md := stack.getManifest(t, metaName)
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		if len(signed.Signatures) != 1 {
			t.Fatalf("missing signatures on %s", metaName)
		}
		if !pubKey.ContainsID(signed.Signatures[0].KeyID) {
			t.Fatalf("missing key id for signer on %s", metaName)
		}
		if len(signed.Signatures[0].Signature) == 0 {
			t.Fatalf("missing signature on %s", metaName)
		}
	}
}

func TestSnapshotUnvalidatedFails(t *testing.T) {
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)
	_ = stack.genKey(t, true)

	// Initialize with threshold 1 succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Validate that root and targets have two unfilled signature.
	for _, metaName := range []string{"root.json", "targets.json"} {
		md := stack.getManifest(t, metaName)
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		if len(signed.Signatures) != 2 {
			t.Fatalf("expected 2 signature on %s", metaName)
		}
		if len(signed.Signatures[0].Signature) != 0 {
			t.Fatalf("expected empty signature for key ID %s", signed.Signatures[0].KeyID)
		}
	}

	// Try to snapshot. Expect to fail.
	if err := app.SnapshotCmd(ctx, stack.repoDir); err == nil {
		t.Fatalf("expected Snapshot command to fail")
	}

	// Now sign root and targets with 1/1 threshold key.
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Expect that there is still one empty placeholder signature.
	for _, metaName := range []string{"root.json", "targets.json"} {
		md := stack.getManifest(t, metaName)
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
	stack.snapshot(t)
	for _, metaName := range []string{"root.json", "targets.json"} {
		md := stack.getManifest(t, metaName)
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			t.Fatal(err)
		}
		if len(signed.Signatures) != 1 {
			t.Fatalf("expected 1 signature on %s, got %d", metaName, len(signed.Signatures))
		}
	}
}

func TestPublishSuccess(t *testing.T) {
	// Initialize.
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)

	// Initialize with 1 succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Check versions.
	for _, metaName := range []string{"root.json", "targets.json", "snapshot.json", "timestamp.json"} {
		md := stack.getManifest(t, metaName)
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
	targetFiles, err := verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json"))
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
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef1 := stack.genKey(t, true)
	rootKeyRef2 := stack.genKey(t, true)

	// Initialize succeeds
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets with key 1
	rootSigner1 := stack.getSigner(t, rootKeyRef1)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner1, false); err != nil {
		t.Fatal(err)
	}
	rootTufKey1, err := keys.ConstructTufKey(ctx, rootSigner1)
	if err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Check that there are two root key signers: key 1 and key 2.
	store := tuf.FileSystemStore(stack.repoDir, nil)
	root, err := prepo.GetRootFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	rootRole, ok := root.Roles["root"]
	if !ok {
		t.Fatalf("expected root role")
	}
	rootSigner2 := stack.getSigner(t, rootKeyRef2)
	rootTufKey2, err := keys.ConstructTufKey(ctx, rootSigner2)
	if err != nil {
		t.Fatal(err)
	}
	expectedKeyIds := append(rootTufKey1.IDs(), rootTufKey2.IDs()...)
	actualKeyIds := rootRole.KeyIDs
	sort.Strings(expectedKeyIds)
	sort.Strings(actualKeyIds)
	if !cmp.Equal(expectedKeyIds, actualKeyIds) {
		t.Fatalf("expected key IDs %s, got %s", expectedKeyIds, actualKeyIds)
	}

	// Now remove the second key and add a third key.
	stack.removeHsmKey(t, rootKeyRef2)
	rootKeyRef3 := stack.genKey(t, true)

	// Create a new root.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
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
	rootSigner3 := stack.getSigner(t, rootKeyRef3)
	rootTufKey3, err := keys.ConstructTufKey(ctx, rootSigner3)
	if err != nil {
		t.Fatal(err)
	}
	expectedKeyIds = append(rootTufKey1.IDs(), rootTufKey3.IDs()...)
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
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"}, rootSigner1,
		false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Verify with go-tuf
	if _, err = verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRotateTarget(t *testing.T) {
	// Initialize.
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Check versions.
	checkMetadataVersion(t, stack.repoDir,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		1)

	// Verify with go-tuf
	targetFiles, err := verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json"))
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
	stack.removeTarget(t, "foo.txt")
	stack.addTarget(t, "bar.txt", "abcdef", nil)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Check versions.
	checkMetadataVersion(t, stack.repoDir,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		2)

	// Verify with go-tuf
	targetFiles, err = verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json"))
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
	// Initialize.
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)

	// Initialize succeeds with consistent snapshot off.
	app.ConsistentSnapshot = false
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Check versions.
	checkMetadataVersion(t, stack.repoDir,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		1)

	// Verify with go-tuf
	targetFiles, err := verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json"))
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
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}
	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Check versions.
	checkMetadataVersion(t, stack.repoDir,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		2)

	// Verify with TUF
	targetFiles, err = verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json"))
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

	// Verify consistent snapshotting was enabled by
	// checking that 2.snapshot.json is present.
	repoFiles, err := ioutil.ReadDir(filepath.Join(stack.repoDir, "repository"))
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

// Tests snapshot key rotation.
func TestSnapshotKeyRotate(t *testing.T) {
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets with key 1
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Verify that the snapshot role contains the initial snapshot key id
	store := tuf.FileSystemStore(stack.repoDir, nil)
	root, err := prepo.GetRootFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	snapshotRole, ok := root.Roles["snapshot"]
	if !ok {
		t.Fatalf("expected snapshot role")
	}
	snapshotSigner1 := stack.getSigner(t, stack.snapshotRef)
	snapshotTufKey1, err := keys.ConstructTufKey(ctx, snapshotSigner1)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshotRole.KeyIDs) != 1 {
		t.Errorf("expected one snapshot key")
	}
	if snapshotRole.KeyIDs[0] != snapshotTufKey1.IDs()[0] {
		t.Errorf("expected snapshot key %s, got %s", snapshotTufKey1.IDs()[0],
			snapshotRole.KeyIDs[0])
	}

	// Now rotate the snapshot signer out.
	stack.snapshotRef = createTestSigner(t)
	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets with key 1
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Expect only the new snapshot key.
	root, err = prepo.GetRootFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	snapshotRole, ok = root.Roles["snapshot"]
	if !ok {
		t.Fatalf("expected snapshot role")
	}
	snapshotSigner2 := stack.getSigner(t, stack.snapshotRef)
	snapshotTufKey2, err := keys.ConstructTufKey(ctx, snapshotSigner2)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshotRole.KeyIDs) != 1 {
		t.Errorf("expected one snapshot key")
	}
	if snapshotRole.KeyIDs[0] != snapshotTufKey2.IDs()[0] {
		t.Errorf("expected snapshot key %s, got %s", snapshotTufKey2.IDs()[0],
			snapshotRole.KeyIDs[0])
	}
}

func TestProdTargetsConfig(t *testing.T) {
	// Initialize.
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	rootKeyRef := stack.genKey(t, true)

	wd, _ := os.Getwd()
	configBytes, err := ioutil.ReadFile(
		filepath.Join(wd, "../config/targets-metadata.yml"))
	if err != nil {
		t.Fatal(err)
	}
	targetsDir := filepath.Join(wd, "../targets")
	targetsConfig, err := prepo.SigstoreTargetMetaFromString(configBytes)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		targetsConfig, targetsDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Check versions.
	checkMetadataVersion(t, stack.repoDir,
		[]string{"root.json", "targets.json", "snapshot.json", "timestamp.json"},
		1)

	// Verify with go-tuf
	targetFiles, err := verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(targetFiles) != len(targetsConfig.Add) {
		t.Fatalf("expected %d target, got %d", len(targetsConfig.Add), len(targetFiles))
	}
	// Validate presence of custom metadata per configuration.
	for name, tFiles := range targetFiles {
		var v1, v2 interface{}
		json.Unmarshal([]byte(targetsConfig.Add[name]), &v1)
		if tFiles.Custom == nil {
			if v2 != nil {
				t.Errorf("no custom metadata found, expected: %s", v2)
			}
			continue
		}
		json.Unmarshal([]byte(*tFiles.Custom), &v2)
		if !reflect.DeepEqual(v1, v2) {
			t.Errorf("expected custom %s, got %s", targetsConfig.Add[name], *tFiles.Custom)
		}
	}

	// Verify the expiration of targets
	store := tuf.FileSystemStore(stack.repoDir, nil)
	targets, err := prepo.GetTargetsFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	if targets.Expires.Sub(app.GetExpiration("targets")).Round(time.Hour) != 0 {
		t.Errorf("expected expiration %s", app.GetExpiration("targets"))
	}
}

// Tests that initializing a new root and targets leaves targets
// in a clear state.
func TestDelegationsClearedOnInit(t *testing.T) {
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets with key 1
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Verify that targets does not have any delegations.
	store := tuf.FileSystemStore(stack.repoDir, nil)
	targets, err := prepo.GetTargetsFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	if targets.Delegations != nil {
		t.Errorf("Expected top-level targets delegations to be cleared")
	}
}

func TestSignWithVersionBump(t *testing.T) {
	ctx := context.Background()
	stack := newRepoTestStack(ctx, t)
	stack.addTarget(t, "foo.txt", "abc", nil)
	rootKeyRef := stack.genKey(t, true)

	// Initialize succeeds.
	if err := app.InitCmd(ctx, stack.repoDir, 1,
		stack.targetsConfig, stack.repoDir, stack.snapshotRef, stack.timestampRef); err != nil {
		t.Fatal(err)
	}

	// Add a delegation
	delegationKeyRef, delegationPubKeyRef := createTestSignVerifier(t)
	if err := app.DelegationCmd(ctx,
		&app.DelegationOptions{
			Directory:   stack.repoDir,
			Name:        "delegation",
			Path:        "path/*",
			Terminating: true,
			KeyRefs:     []string{delegationPubKeyRef},
			Threshold:   1,
			Targets:     "",
		}); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets with key 1
	rootSigner := stack.getSigner(t, rootKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"root", "targets"},
		rootSigner, false); err != nil {
		t.Fatal(err)
	}
	// Sign delegation
	dSigner := stack.getSigner(t, delegationKeyRef)
	if err := app.SignCmd(ctx, stack.repoDir, []string{"delegation"},
		dSigner, false); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	if _, err := verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json")); err != nil {
		t.Fatal(err)
	}
	checkMetadataVersion(t, stack.repoDir, []string{"delegation.json"}, 1)

	// Increment the delegation metadata
	if err := app.SignCmd(ctx, stack.repoDir, []string{"delegation"},
		dSigner, true); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	stack.snapshot(t)
	stack.timestamp(t)
	stack.publish(t)

	// Verify with go-tuf
	if _, err := verifyTuf(t, stack.repoDir, stack.getManifest(t, "root.json")); err != nil {
		t.Fatal(err)
	}

	// Check delegation version bump
	checkMetadataVersion(t, stack.repoDir, []string{"delegation.json"}, 2)
}

func TestKeyPOP(t *testing.T) {
	ctx := context.Background()
	keyRef, pubKeyRef := createTestSignVerifier(t)
	challenge := "some data"
	nonce := "not random at all"

	verifier, err := app.GetVerifier(ctx, pubKeyRef)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := app.GetSigner(ctx, false, keyRef)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := app.DoKeyPOPSign(ctx, challenge, nonce, signer)
	if err != nil {
		t.Fatal(err)
	}
	err = app.KeyPOPVerifyCmd(ctx, challenge, nonce, verifier, sig)
	if err != nil {
		t.Fatal(err)
	}

	// Verify that we got proper protection against canonicalizaton
	// attacks
	challenge = "some dat"
	nonce = "anot random at all"
	err = app.KeyPOPVerifyCmd(ctx, challenge, nonce, verifier, sig)
	if err == nil {
		t.Fatal("verification should fail")
	}
}
