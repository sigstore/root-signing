//go:build pivkey
// +build pivkey

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/root-signing/cmd/tuf/app"
	vapp "github.com/sigstore/root-signing/cmd/verify/app"
	"github.com/sigstore/root-signing/pkg/keys"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
)

// TODO(asraa): Add more unit tests, including
//   * Custom metadata included in targets
//   * Updating a root version
//   * Rotating a keyholder
//   * Fetching targets with cosign's API with/without consistent snapshotting
//   * Rotate a target file

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
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey); err != nil {
		t.Fatal(err)
	}

	// Verify that root and targets have expected version 1 on Init.
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
		sm, err := vapp.PrintAndGetSignedMeta(metaName, signed.Signed)
		if err != nil {
			t.Fatal(err)
		}
		if sm.Version != 1 {
			t.Errorf("expected root version 1, got %d", sm.Version)
		}
	}
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
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey); err != nil {
		t.Fatal(err)
	}

	// Sign root and targets.
	signerAndKey, err := GetTestHsmSigner(ctx, td, *serial)
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
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey); err != nil {
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
	signerAndKey1, err := GetTestHsmSigner(ctx, td, *rootkey1)
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
	if err := app.InitCmd(ctx, td, "", 1, targetsConfig, snapshotKey, timestampKey); err != nil {
		t.Fatal(err)
	}

	// Sign root & targets
	rootKey, err := GetTestHsmSigner(ctx, td, *rootSerial)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"root", "targets"}, rootKey); err != nil {
		t.Fatal(err)
	}

	// Sign snapshot and timestamp
	if err := app.SnapshotCmd(ctx, td); err != nil {
		t.Fatalf("expected Snapshot command to pass, got err: %s", err)
	}
	snapshotSigner, err := keys.GetSigningKey(ctx, snapshotKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"snapshot"}, snapshotSigner); err != nil {
		t.Fatal(err)
	}

	if err := app.TimestampCmd(ctx, td); err != nil {
		t.Fatalf("expected Timestamp command to pass, got err: %s", err)
	}
	timestampSigner, err := keys.GetSigningKey(ctx, timestampKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(ctx, td, []string{"timestamp"}, timestampSigner); err != nil {
		t.Fatal(err)
	}

	// Successful Publishing!
	if err := app.PublishCmd(ctx, td); err != nil {
		t.Fatal(err)
	}

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
	remote, err := vapp.FileRemoteStore(td)
	if err != nil {
		t.Fatal(err)
	}
	local := client.MemoryLocalStore()
	c := client.NewClient(local, remote)
	if err := c.InitLocal(meta["root.json"]); err != nil {
		t.Fatal(err)

	}
	targetFiles, err := c.Update()
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
