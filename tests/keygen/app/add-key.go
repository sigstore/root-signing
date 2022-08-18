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

package app

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	test "github.com/sigstore/root-signing/tests"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

var (
	repository   string
	caPath       string
	caSignerPath string
)

var addKeyCmd = &cobra.Command{
	Use:   "add-key",
	Short: "Test add key command",
	Long:  `Generates a test HSM key`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error initializing cmd line args: %s", err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.SetFlags(0)

		rootCaBytes, err := ioutil.ReadFile(caPath)
		if err != nil {
			log.Fatal(err)
		}

		rootCAs, err := cryptoutils.UnmarshalCertificatesFromPEM(rootCaBytes)
		if err != nil {
			log.Fatal(err)
		}
		if len(rootCAs) != 1 {
			log.Fatal(fmt.Errorf("missing certificate for root"))
		}

		rootSignerBytes, err := ioutil.ReadFile(caSignerPath)
		if err != nil {
			log.Fatal(err)
		}

		rootSigner, err := cryptoutils.UnmarshalPEMToPrivateKey(rootSignerBytes, nil)
		if err != nil {
			log.Fatal(err)
		}

		if _, err := test.CreateTestHsmSigner(repository, rootCAs[0], rootSigner); err != nil {
			log.Fatal(err)
		}

	},
}

func init() {
	addKeyCmd.Flags().StringVar(&caPath, "root", "test-root-attestation.ca", "path to root certificate")
	addKeyCmd.Flags().StringVar(&caSignerPath, "root-signer", "test-root.pem", "path to root signer, PEM encoded private key")
	addKeyCmd.Flags().StringVar(&repository, "repository", "repository/", "path to repository to write key")

	rootCmd.AddCommand(addKeyCmd)
}
