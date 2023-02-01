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
	"encoding/base64"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/options"
)

func KeyPOPVerify() *ffcli.Command {
	var (
		flagset   = flag.NewFlagSet("tuf key-pop-sign", flag.ExitOnError)
		challenge = flagset.String("challenge", "", "the challenge to sign, for a delegate this is the delegate name")
		nonce     = flagset.String("nonce", "", "the nonce delivered out of band to the key holder")
		key       = flagset.String("key", "", "reference to a signer for signing")
		sig       = flagset.String("sig", "", "base64 encoded signature to verify")
	)

	return &ffcli.Command{
		Name:       "key-pop-verify",
		ShortUsage: "tuf key-pop-verify -challenge value -nonce nonce -key ref -sig base64e",
		ShortHelp:  "Verify a proof of possession of a key",
		LongHelp:   "Verify a proof of possession of a key",
		FlagSet:    flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *challenge == "" {
				return flag.ErrHelp
			}
			if *nonce == "" {
				return flag.ErrHelp
			}
			if *key == "" {
				return flag.ErrHelp
			}
			if *sig == "" {
				return flag.ErrHelp
			}

			verifier, err := GetVerifier(ctx, *key)
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
