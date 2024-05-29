//
// Copyright 2023 The Sigstore Authors.
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
	"context"
	"crypto"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sigstore/root-signing/pkg/keys"
	"github.com/sigstore/root-signing/pkg/repo"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/options"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
)

const topLevelTargetsFilename = "targets.json"

func KeyPOPVerify() *ffcli.Command {
	var (
		flagset    = flag.NewFlagSet("tuf key-pop-sign", flag.ExitOnError)
		challenge  = flagset.String("challenge", "", "the challenge to sign, for a delegate this is the delegate name")
		nonce      = flagset.String("nonce", "", "the nonce delivered out of band to the key holder")
		keyid      = flagset.String("keyid", "", "key id fo the the delegation")
		sig        = flagset.String("sig", "", "base64 encoded signature to verify")
		role       = flagset.String("role", "", "delegation to verify")
		repository = flagset.String("repository", "", "path to the staged repository")
	)

	return &ffcli.Command{
		Name:       "key-pop-verify",
		ShortUsage: "tuf key-pop-verify -challenge value -nonce nonce -key ref -sig base64e",
		ShortHelp:  "Verify a proof of possession of a key",
		LongHelp:   "Verify a proof of possession of a key",
		FlagSet:    flagset,
		Exec: func(ctx context.Context, _ []string) error {
			if *challenge == "" {
				return flag.ErrHelp
			}
			if *nonce == "" {
				return flag.ErrHelp
			}
			if (*keyid == "") == (*role == "") {
				return flag.ErrHelp
			}
			if *sig == "" {
				return flag.ErrHelp
			}
			if *repository == "" {
				return flag.ErrHelp
			}

			if *role != "" {
				inferredKey, err := GetKeyIDForRole(*repository, *role)
				if err != nil {
					return err
				}
				keyid = &inferredKey
				fmt.Fprintf(os.Stderr, "Verifying using keyid %s\n", *keyid)
			}
			pubKey, err := GetPublicKeyFromID(*repository, *keyid)
			if err != nil {
				return err
			}

			verifier, err := signature.LoadVerifier(pubKey, crypto.SHA256)
			if err != nil {
				return err
			}
			sigBytes, err := base64.StdEncoding.DecodeString(*sig)
			if err != nil {
				return err
			}

			return KeyPOPVerifyCmd(ctx, *challenge, *nonce, verifier, sigBytes)
		},
	}
}

func GetKeyIDForRole(directory, role string) (string, error) {
	store := tuf.FileSystemStore(directory, nil)
	signed, err := repo.GetSignedMeta(store, topLevelTargetsFilename)
	if err != nil {
		return "", err
	}
	meta, err := repo.GetMetaFromStore(signed.Signed, topLevelTargetsFilename)
	if err != nil {
		return "", err
	}
	targets := meta.(*data.Targets)

	for _, delegatedRole := range targets.Delegations.Roles {
		if delegatedRole.Name != role {
			continue
		}

		if len(delegatedRole.KeyIDs) != 1 {
			return "", fmt.Errorf("found %d keys for role %s, expected 1",
				len(delegatedRole.KeyIDs), role)
		}

		return delegatedRole.KeyIDs[0], nil
	}

	return "", fmt.Errorf("unknown delegation %s", role)
}

func GetPublicKeyFromID(directory, keyid string) (crypto.PublicKey, error) {
	store := tuf.FileSystemStore(directory, nil)
	signed, err := repo.GetSignedMeta(store, topLevelTargetsFilename)
	if err != nil {
		return nil, err
	}
	meta, err := repo.GetMetaFromStore(signed.Signed, topLevelTargetsFilename)
	if err != nil {
		return nil, err
	}
	targets := meta.(*data.Targets)

	candidateKey, ok := targets.Delegations.Keys[keyid]
	if !ok {
		return nil, fmt.Errorf("unknown key %s", keyid)
	}

	var keyValue keys.KeyValue
	err = keyValue.Unmarshal(candidateKey)
	if err != nil {
		return nil, err
	}

	pemBytes := keyValue.PublicKey
	pemBytes = strings.ReplaceAll(pemBytes, "\\n", "\n")

	return cryptoutils.UnmarshalPEMToPublicKey([]byte(pemBytes))
}

func KeyPOPVerifyCmd(ctx context.Context,
	challenge, nonce string,
	verifier signature.Verifier,
	sig []byte) error {
	data := PAE(challenge, nonce)

	err := verifier.VerifySignature(bytes.NewReader(sig),
		bytes.NewReader(data),
		options.WithContext(ctx))
	if err != nil {
		return err
	}

	fmt.Println("Signature verified ok")

	return nil
}
