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

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sigstore/root-signing/pkg/repo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/verify"
)

type bufferDestination struct {
	bytes.Buffer
	deleted bool
}

func (t *bufferDestination) Delete() error {
	t.deleted = true
	return nil
}

type fileRemoteStore struct {
	Repo string
	Meta map[string]json.RawMessage
}

func FileRemoteStore(repo string) (client.RemoteStore, error) {
	// Load all the metadata only from the committed repository/
	repoDir := filepath.Join(repo, "repository")
	committed, err := os.ReadDir(repoDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("could not list repo dir: %w", err)
	}

	meta := make(map[string]json.RawMessage)
	for _, e := range committed {
		imf, err := isMetaFile(e)
		if err != nil {
			return nil, err
		}
		if imf {
			name := e.Name()
			f, err := os.ReadFile(filepath.Join(repoDir, name))
			if err != nil {
				return nil, err
			}
			meta[name] = f
		}
	}

	return fileRemoteStore{Repo: repo, Meta: meta}, nil
}

func (r fileRemoteStore) GetMeta(name string) (io.ReadCloser, int64, error) {
	meta, ok := r.Meta[name]
	if !ok {
		return nil, 0, client.ErrNotFound{File: name}
	}
	return io.NopCloser(bytes.NewReader(meta)), int64(len(meta)), nil
}

func (r fileRemoteStore) GetTarget(target string) (io.ReadCloser, int64, error) {
	payload, err := os.ReadFile(filepath.Join(r.Repo, "repository", "targets", target))
	if err != nil {
		return nil, 0, err
	}
	return io.NopCloser(bytes.NewReader(payload)), int64(len(payload)), nil
}

// Metadata helpers
type signedMeta struct {
	Type    string    `json:"_type"`
	Expires time.Time `json:"expires"`
	Version int       `json:"version"`
}

func isMetaFile(e os.DirEntry) (bool, error) {
	if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
		return false, nil
	}

	info, err := e.Info()
	if err != nil {
		return false, err
	}

	return info.Mode().IsRegular(), nil
}

func PrintAndGetSignedMeta(role string, signed json.RawMessage) (*signedMeta, error) {
	sm := &signedMeta{}
	if err := json.Unmarshal(signed, sm); err != nil {
		return nil, err
	}
	fmt.Printf("\t%s version %d, expires %s\n", role, sm.Version, sm.Expires.Format("2006/01/02"))
	return sm, nil
}

func verifyStagedMetadata(repository string) error {
	log.Printf("\nOutputting metadata verification at %s...\n", repository)

	// logs the state of each metadata file, including number of signatures to achieve threshold
	// and verifies the signatures in each file.
	store := tuf.FileSystemStore(repository, nil)

	meta, err := store.GetMeta()
	if err != nil {
		return err
	}

	db, thresholds, err := repo.CreateDb(store)
	if err != nil {
		return err
	}

	// Create a DB with just the previous root for root verification against
	// the previous keys
	prevRootExists := false
	var prevDb *verify.DB
	var prevThresholds map[string]int
	prevRoot, err := repo.GetPreviousRootFromStore(store)
	if err != nil {
		if !errors.Is(err, repo.ErrNoPreviousRoot) {
			return fmt.Errorf("error getting previous root: %w", err)
		}
	} else {
		prevRootExists = true
		prevRootBytes, ok := meta[fmt.Sprintf("%d.root.json", int(prevRoot.Version))]
		if !ok {
			// This is an error, we should have a previous root.
			return fmt.Errorf("expected %d.root.json in store", prevRoot.Version)
		}
		var err error
		prevDb, prevThresholds, err = repo.CreateDb(
			tuf.MemoryStore(map[string]json.RawMessage{
				"root.json": prevRootBytes}, nil))
		if err != nil {
			return err
		}
	}

	for name, md := range meta {
		if repo.IsVersionedManifest(name) {
			continue
		}
		// only verify staged.
		if !store.FileIsStaged(name) {
			continue
		}

		log.Printf("\nVerifying %s...", name)

		name = strings.TrimSuffix(name, ".json")
		signed := &data.Signed{}
		if err := json.Unmarshal(md, signed); err != nil {
			return err
		}

		// Remove the empty placeholder signatures
		var sigs []data.Signature
		for _, sig := range signed.Signatures {
			if len(sig.Signature) != 0 {
				sigs = append(sigs, sig)
			}
		}
		signed.Signatures = sigs

		validSigs, err := verifySignatures(db, signed, name)
		if err != nil {
			log.Printf("\tError verifying: %s\n", err)
			return err
		}
		if validSigs == thresholds[name] {
			log.Printf("\tSuccess! Signatures valid and threshold achieved\n")
		} else {
			log.Printf("\tContains %d/%d valid signatures from the current staged metadata\n", validSigs, thresholds[name])
		}

		if prevRootExists && name == "root" {
			// Also verify from the previous root for the root role.
			validSigs, err := verifySignatures(prevDb, signed, name)
			if err != nil {
				log.Printf("\tError verifying: %s\n", err)
				return err
			}
			if validSigs == prevThresholds[name] {
				log.Printf("\tSuccess! Signatures valid and threshold achieved from the previous root\n")
			} else {
				log.Printf("\tContains %d/%d valid signatures from the previous root\n", validSigs, prevThresholds[name])
			}
		}

		if _, err := PrintAndGetSignedMeta(name, signed.Signed); err != nil {
			return err
		}
	}

	return nil
}

func verifySignatures(db *verify.DB, s *data.Signed, role string) (int, error) {
	err := db.VerifySignatures(s, role)
	if _, ok := err.(verify.ErrRoleThreshold); ok {
		return err.(verify.ErrRoleThreshold).Actual, nil
	} else if errors.Is(err, verify.ErrNoSignatures) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	// We have success, a threshold number of signers signed.
	dbRole := db.GetRole(role)
	return dbRole.Threshold, nil
}

func getClientState(local client.LocalStore) (map[string]signedMeta, error) {
	trustedMeta, err := local.GetMeta()
	res := make(map[string]signedMeta, len(trustedMeta))
	if err != nil {
		return nil, errors.Wrap(err, "getting trusted meta")
	}

	for role, md := range trustedMeta {
		s := &data.Signed{}
		if err := json.Unmarshal(md, s); err != nil {
			return nil, err
		}
		sm, err := PrintAndGetSignedMeta(role, s.Signed)
		if err != nil {
			return nil, err
		}
		res[role] = *sm
	}

	return res, nil
}

var (
	repository string
	staged     bool
	root       file
	expiration string
	roles      []string
)

var repositoryCmd = &cobra.Command{
	Use:   "repository",
	Short: "Root verify repository command",
	Long:  `Verifies repository metadata and prints targets retrieved`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error initializing cmd line args: %s", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(0)

		if !staged && root.String() == "" {
			log.Println("must specify a trusted root file for full verification")
			_ = cmd.Usage()
			os.Exit(1)
		}

		var validUntil *time.Time
		if expiration != "" {
			parsedTime, err := time.Parse("2006/01/02", expiration)
			if err != nil {
				log.Printf("must specify a valid time, got %s", expiration)
				_ = cmd.Usage()
				os.Exit(1)
			}
			validUntil = &parsedTime
		}

		if err := VerifyCmd(staged, repository, root.String(), validUntil, roles); err != nil {
			log.Println(err.Error())
		}
	},
}

func checkRoleExpiry(role string, toCheck []string) bool {
	if roles == nil {
		return true
	}
	for _, r := range toCheck {
		if r == role {
			return true
		}
	}
	return false
}

func init() {
	repositoryCmd.Flags().StringArrayVar(&roles, "role", nil, "set multiple times for multiple roles (all roles verified if unset)")
	repositoryCmd.Flags().StringVar(&repository, "repository", "repository/", "path to repository, may be HTTP or local file")
	repositoryCmd.Flags().BoolVar(&staged, "staged", false, "indicates whether the repository is staged and should only be partially verified")
	repositoryCmd.Flags().Var(&root, "root", "path to a trusted root, required unless verifying staged metadata")
	repositoryCmd.Flags().StringVar(&expiration, "valid-until", "", "a time for metadata to be valid until e.g. 2022/02/22")
	_ = repositoryCmd.MarkFlagRequired("repository")

	rootCmd.AddCommand(repositoryCmd)
}

func VerifyCmd(staged bool, repository string, rootFile string,
	validUntil *time.Time, rolesToCheck []string) error {
	if staged {
		// Assumes a local repository!
		// This will include staged metadata and verify partial signatures
		log.Printf("STAGED METADATA")
		if err := verifyStagedMetadata(repository); err != nil {
			return fmt.Errorf("error verifying metadata: %s", err)
		}
		return nil
	}

	// Otherwise verify a repository in full.
	log.Printf("\nVERIFYING TUF CLIENT UPDATE\n\n")

	rootMeta, err := os.ReadFile(rootFile)
	if err != nil {
		return fmt.Errorf("error reading trusted TUF root %s: %w", root, err)
	}
	local := client.MemoryLocalStore()
	if err := local.SetMeta("root.json", rootMeta); err != nil {
		return fmt.Errorf("error setting root metadata: %s", err)
	}

	var remote client.RemoteStore
	u, err := url.ParseRequestURI(repository)
	if err != nil {
		return fmt.Errorf("error parsing remote repository location: %s", err)
	}
	if u.IsAbs() {
		remote, err = client.HTTPRemoteStore(repository, nil, nil)
		if err != nil {
			return fmt.Errorf("error reading trusted TUF HTTP remote: %s", err)
		}
	} else {
		remote, err = FileRemoteStore(repository)
		if err != nil {
			return fmt.Errorf("error reading trusted TUF local file remote: %s", err)
		}
	}
	c := client.NewClient(local, remote)

	if err := c.Init(rootMeta); err != nil {
		return fmt.Errorf("error initializing client: %s", err)
	}

	// TODO: Update only returns top-level targets!
	log.Printf("Client successfully initialized, updating and downloading targets...")
	targetFiles, err := c.Update()
	if err != nil {
		return fmt.Errorf("error updating client: %s", err)
	}
	log.Printf("Client updated to...")
	clientState, err := getClientState(local)
	if err != nil {
		return fmt.Errorf("error getting client state: %s", err)
	}
	for name := range targetFiles {
		var dest bufferDestination
		if err := c.Download(name, &dest); err != nil {
			return fmt.Errorf("error downloading target: %s", err)
		}
		log.Printf("\nRetrieved target %s...", name)
		log.Printf("%s", dest.Bytes())
	}

	// If verified, check expiration
	if validUntil != nil {
		for role, sm := range clientState {
			if !checkRoleExpiry(role, rolesToCheck) {
				continue
			}
			if sm.Expires.Before(*validUntil) {
				return fmt.Errorf("error: %s will expire on %s\n", role, sm.Expires.Format("2006/01/02"))
			}
		}
	}
	return nil
}
