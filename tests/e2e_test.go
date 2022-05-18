package test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/root-signing/cmd/tuf/app"
	vapp "github.com/sigstore/root-signing/cmd/verify/app"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
)

// Create test HSM keys.
// This function creates a test temp directory of fake hardware keys.
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
	path := path.Join(temp, "cosign.key")

	if err := os.WriteFile(path, keys.PrivateBytes, 0600); err != nil {
		t.Fatal(err)
	}
	return path
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
