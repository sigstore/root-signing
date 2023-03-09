//
// Copyright 2021 The Sigstore Authors.
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
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/pivcli"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"golang.org/x/term"
)

func AddKey() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf add-key", flag.ExitOnError)
		repository = flagset.String("repository", "", "path to the staged repository")
	)
	return &ffcli.Command{
		Name:       "add-key",
		ShortUsage: "tuf add-key adds a new root key to the given repository",
		ShortHelp:  "tuf add-key adds a new root key to the given repository",
		LongHelp: `tuf add-key adds a new root key to the given repository.
		It adds them to the {root, targets} top-level roles. 
		TODO: When keyval supports a custom JSON, add it certs to the JSON.
		
	EXAMPLES
	# add-key to staged repository at ceremony/YYYY-MM-DD
	tuf add-key -repository ceremony/YYYY-MM-DD`,
		FlagSet: flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *repository == "" {
				return flag.ErrHelp
			}
			return AddKeyCmd(ctx, *repository)
		},
	}
}

type KeyAndAttestations struct {
	Attestations pivcli.Attestations
	Key          *ecdsa.PublicKey
}

func GetKeyAndAttestation(ctx context.Context) (*KeyAndAttestations, error) {
	attestations, err := pivcli.AttestationCmd(ctx, "signature")
	if err != nil {
		return nil, err
	}

	pub := attestations.KeyCert.PublicKey.(*ecdsa.PublicKey)

	return &KeyAndAttestations{Attestations: *attestations, Key: pub}, nil
}

func AddKeyCmd(ctx context.Context, directory string) error {
	if err := pivcli.ResetKeyCmd(ctx); err != nil {
		return err
	}

	if err := pivcli.GenerateKeyCmd(ctx, "", true, "signature", "always", "always"); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Resetting PIN. Enter a new PIN between 6 and 8 characters: \n")
	pin, err := term.ReadPassword(0)
	if err != nil {
		return err
	}
	if err := pivcli.SetPinCmd(ctx, "", string(pin)); err != nil {
		return err
	}

	keyAndAttestations, err := GetKeyAndAttestation(ctx)
	if err != nil {
		return err
	}

	// Write to repository/keys/SERIAL_NUM/SERIAL_NUM_pubkey.pem, etc
	return WriteKeyData(keyAndAttestations, directory)
}

func WriteKeyData(keyAndAttestations *KeyAndAttestations, directory string) error {
	att := keyAndAttestations.Attestations
	serial := fmt.Sprint(att.KeyAttestation.Serial)
	keyDir := filepath.Join(directory, "keys", serial)
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return err
	}

	pemBytes, err := cryptoutils.MarshalPublicKeyToPEM(keyAndAttestations.Attestations.KeyCert.PublicKey)
	if err != nil {
		return err
	}
	pubKeyFile := filepath.Join(keyDir, serial+"_pubkey.pem")
	if err := ioutil.WriteFile(pubKeyFile, pemBytes, 0644); err != nil {
		return err
	}
	keyCertFile := filepath.Join(keyDir, serial+"_key_cert.pem")
	if err := ioutil.WriteFile(keyCertFile, []byte(att.KeyCertPem), 0644); err != nil {
		return err
	}
	deviceCertFile := filepath.Join(keyDir, serial+"_device_cert.pem")
	if err := ioutil.WriteFile(deviceCertFile, []byte(att.DeviceCertPem), 0644); err != nil {
		return err
	}
	fmt.Println("Wrote public key data to ", keyDir)

	return nil
}
