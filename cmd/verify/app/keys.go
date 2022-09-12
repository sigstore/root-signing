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
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sigstore/root-signing/pkg/keys"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type file string

func (f *file) String() string {
	return string(*f)
}

func (f *file) Type() string {
	return "file"
}

func (f *file) Set(s string) error {
	if s == "" {
		return errors.New("flag must be specified")
	}
	if _, err := os.Stat(filepath.Clean(s)); os.IsNotExist(err) {
		return err
	}
	*f = file(s)
	return nil
}

func toCert(filename string) (*x509.Certificate, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	certs, err := cryptoutils.UnmarshalCertificatesFromPEMLimited(
		fileBytes, 2)
	if err != nil {
		return nil, err
	}
	if len(certs) != 1 {
		return nil, errors.New("expected one PEM-encoded certificate")
	}
	return certs[0], nil
}

// Map from Key ID to Signing Key
type KeyMap map[string]*keys.SigningKey

func getKeyID(key keys.SigningKey) (string, error) {
	pk, err := keys.ToTufKey(key, false)
	if err != nil {
		return "", err
	}
	if len(pk.IDs()) == 0 {
		return "", errors.New("error getting key ID")
	}
	return pk.IDs()[0], nil
}

func verifySigningKeys(dirname string, rootCA *x509.Certificate) (*KeyMap, error) {
	// Get all signing keys in the directory.
	log.Printf("\nOutputting key verification and OpenSSL commands...\n")

	files, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	keyMap := make(KeyMap)
	for _, file := range files {
		if file.IsDir() {
			key, err := keys.SigningKeyFromDir(filepath.Join(dirname, file.Name()))
			if err != nil {
				return nil, err
			}
			if err = key.Verify(rootCA); err != nil {
				log.Printf("error verifying key %d: %s", key.SerialNumber, err)
				return nil, err
			}
			id, err := getKeyID(*key)
			if err != nil {
				return nil, err
			}

			log.Printf("\nVERIFIED KEY WITH SERIAL NUMBER %d\n", key.SerialNumber)
			log.Printf("\tTUF key id: %s\n", id)

			keyMap[id] = key
		}
	}
	// Note we use relative path here to simplify things.
	serialID := "${SERIAL_NUMBER}"
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	certPath, err := filepath.Rel(wd, filepath.Join(dirname, serialID, serialID+"_device_cert.pem"))
	if err != nil {
		return nil, err
	}
	keyPath, err := filepath.Rel(wd, filepath.Join(dirname, serialID, serialID+"_key_cert.pem"))
	if err != nil {
		return nil, err
	}
	pubkeyPath, err := filepath.Rel(wd, filepath.Join(dirname, serialID, serialID+"_pubkey.pem"))
	if err != nil {
		return nil, err
	}
	log.Printf("\n# To manually verify the chain for any key ID\n\n")
	log.Printf("\texport SERIAL_NUMBER=${SERIAL_NUMBER}")
	log.Printf("\topenssl verify -verbose -x509_strict -CAfile <(cat piv-attestation-ca.pem %s) %s\n", certPath, keyPath)

	log.Printf("\n# Manually extract the public key for any key ID and match with published\n\n")
	log.Printf("\texport SERIAL_NUMBER=${SERIAL_NUMBER}")
	log.Printf("\topenssl x509 -in %s -pubkey -noout", keyPath)
	log.Printf("\tcat %s", pubkeyPath)

	return &keyMap, nil
}

var (
	rootFile file
	keyDir   file
)

var keyCmd = &cobra.Command{
	Use:   "keys",
	Short: "Root verify keys command",
	Long:  `Verifies hardware keys for a repository`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error initializing cmd line args: %s", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(0)

		rootCA, err := toCert(rootFile.String())
		if err != nil {
			log.Printf("failed to parse root CA: %s", err)
			os.Exit(1)
		}

		if _, err = verifySigningKeys(keyDir.String(), rootCA); err != nil {
			log.Printf("error verifying signing keys: %s", err)
			os.Exit(1)
		}

	},
}

func init() {
	keyCmd.Flags().Var(&rootFile, "root", "Yubico root certificate")
	keyCmd.Flags().Var(&keyDir, "key-directory", "path to keys/ directory")
	_ = keyCmd.MarkFlagRequired("root")
	_ = keyCmd.MarkFlagRequired("key-directory")

	rootCmd.AddCommand(keyCmd)
}
