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
	"io/ioutil"
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
			f, err := ioutil.ReadFile(filepath.Join(repoDir, name))
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
	return ioutil.NopCloser(bytes.NewReader(meta)), int64(len(meta)), nil
}

func (r fileRemoteStore) GetTarget(target string) (io.ReadCloser, int64, error) {
	payload, err := ioutil.ReadFile(filepath.Join(r.Repo, "repository", "targets", target))
	if err != nil {
		return nil, 0, err
	}
	return ioutil.NopCloser(bytes.NewReader(payload)), int64(len(payload)), nil
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

	db, thresholds, err := repo.CreateDb(store)
	if err != nil {
		return err
	}

	meta, err := store.GetMeta()
	if err != nil {
		return err
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

		if err = db.VerifySignatures(signed, name); err != nil {
			if _, ok := err.(verify.ErrRoleThreshold); ok {
				// we may not have all the sig, allow partial sigs for success
				log.Printf("\tContains %d/%d valid signatures\n", err.(verify.ErrRoleThreshold).Actual, thresholds[name])
				_, err := PrintAndGetSignedMeta(name, signed.Signed)
				if err != nil {
					return err
				}
			} else if err.Error() == verify.ErrNoSignatures.Error() {
				// We do not return an error here so we can log unsigned metadata
				log.Printf("\tContains 0/%d valid signatures\n", thresholds[name])
				_, err := PrintAndGetSignedMeta(name, signed.Signed)
				if err != nil {
					return err
				}
			} else {
				log.Printf("\tError verifying: %s\n", err)
				return err
			}
		} else {
			log.Printf("\tSuccess! Signatures valid and threshold achieved\n")
			_, err := PrintAndGetSignedMeta(name, signed.Signed)
			if err != nil {
				return err
			}
		}
	}

	return nil
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

		if staged {
			// Assumes a local repository!
			// This will include staged metadata and verify partial signatures
			log.Printf("STAGED METADATA")
			if err := verifyStagedMetadata(repository); err != nil {
				log.Printf("error verifying metadata: %s", err)
				os.Exit(1)
			}
			return
		}

		// Otherwise verify a repository in full.
		if root.String() == "" {
			log.Println("must specify a trusted root file for full verification")
			_ = cmd.Usage()
			os.Exit(1)
		}

		log.Printf("\nVERIFYING TUF CLIENT UPDATE\n\n")

		rootMeta, err := ioutil.ReadFile(root.String())
		if err != nil {
			log.Printf("error reading trusted TUF root: %s", root.String())
			os.Exit(1)
		}
		local := client.MemoryLocalStore()
		if err := local.SetMeta("root.json", rootMeta); err != nil {
			log.Printf("error setting root metadata: %s", err)
			os.Exit(1)
		}

		var remote client.RemoteStore
		u, err := url.ParseRequestURI(repository)
		if err != nil {
			log.Printf("error parsing remote repository location: %s", err)
			os.Exit(1)
		}
		if u.IsAbs() {
			remote, err = client.HTTPRemoteStore(repository, nil, nil)
			if err != nil {
				log.Printf("error reading trusted TUF HTTP remote: %s", err)
				os.Exit(1)
			}
		} else {
			remote, err = FileRemoteStore(repository)
			if err != nil {
				log.Printf("error reading trusted TUF local file remote: %s", err)
				os.Exit(1)
			}
		}
		c := client.NewClient(local, remote)

		if err := c.InitLocal(rootMeta); err != nil {
			log.Printf("error initializing client: %s", err)
			os.Exit(1)
		}

		log.Printf("Client successfully initialized, updating and downloading targets...")
		targetFiles, err := c.Update()
		if err != nil {
			log.Printf("error updating client: %s", err)
			os.Exit(1)
		}
		log.Printf("Client updated to...")
		clientState, err := getClientState(local)
		if err != nil {
			log.Printf("error getting client state: %s", err)
			os.Exit(1)
		}
		for name := range targetFiles {
			var dest bufferDestination
			if err := c.Download(name, &dest); err != nil {
				log.Printf("error downloading target: %s", err)
				os.Exit(1)
			}
			log.Printf("\nRetrieved target %s...", name)
			log.Printf("%s", dest.Bytes())
		}

		// If verified, check expiration
		if expiration != "" {
			validUntil, err := time.Parse("2006/01/02", expiration)
			if err != nil {
				log.Printf("must specify a valid time, got %s", expiration)
				_ = cmd.Usage()
				os.Exit(1)
			}
			for role, sm := range clientState {
				if !checkRoleExpiry(role) {
					continue
				}
				if sm.Expires.Before(validUntil) {
					fmt.Printf("error: %s will expire on %s\n", role, sm.Expires.Format("2006/01/02"))
					os.Exit(1)
				}
			}
		}
	},
}

func checkRoleExpiry(role string) bool {
	if roles == nil {
		return true
	}
	for _, r := range roles {
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
