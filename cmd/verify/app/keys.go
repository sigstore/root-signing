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

// Map from Key ID to Signing Key
type KeyMap map[string]*keys.SigningKey

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

			for _, bv := range []bool{true, false} {
				tufKey, err := keys.EcdsaTufKey(key.PublicKey, bv)
				if err != nil {
					return nil, err
				}
				if len(tufKey.IDs()) == 0 {
					return nil, errors.New("error getting key ID")
				}
				keyMap[tufKey.IDs()[0]] = key
			}

			log.Printf("\nVERIFIED KEY WITH SERIAL NUMBER %d\n", key.SerialNumber)
			log.Printf("TUF key ids: \n")
			for kid, _ := range keyMap {
				log.Printf("\t%s ", kid)
			}
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
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetFlags(0)

		rootBytes, err := os.ReadFile(rootFile.String())
		if err != nil {
			return fmt.Errorf("failed to read root CA file: %s", err)
		}

		rootCA, err := keys.ToCert(rootBytes)
		if err != nil {
			return fmt.Errorf("failed to parse root CA: %s", err)
		}

		if _, err = verifySigningKeys(keyDir.String(), rootCA); err != nil {
			return fmt.Errorf("error verifying signing keys: %s", err)
		}
		return nil
	},
}

func init() {
	keyCmd.Flags().Var(&rootFile, "root", "Yubico root certificate")
	keyCmd.Flags().Var(&keyDir, "key-directory", "path to keys/ directory")
	_ = keyCmd.MarkFlagRequired("root")
	_ = keyCmd.MarkFlagRequired("key-directory")

	rootCmd.AddCommand(keyCmd)
}
