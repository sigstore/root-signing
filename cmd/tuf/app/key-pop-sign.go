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

func KeyPOPSign() *ffcli.Command {
	var (
		flagset   = flag.NewFlagSet("tuf key-pop-sign", flag.ExitOnError)
		challenge = flagset.String("challenge", "", "the challenge to sign, for a delegate this is the delegate name")
		nonce     = flagset.String("nonce", "", "the nonce delivered out of band to the key holder")
		key       = flagset.String("key", "", "reference to a signer for signing")
		sk        = flagset.Bool("sk", false, "indicates use of a hardware key for signing")
	)

	return &ffcli.Command{
		Name:       "key-pop-sign",
		ShortUsage: "tuf key-pop-sign -challenge value -nonce nonce -key ref",
		ShortHelp:  "Sign a proof of possession of a key",
		LongHelp:   "Sign a proof of possession of a key. The base64 encoded signature is printed to stdout",
		FlagSet:    flagset,
		Exec: func(ctx context.Context, args []string) error {
			if *challenge == "" {
				return flag.ErrHelp
			}
			if *nonce == "" {
				return flag.ErrHelp
			}
			if !*sk && *key == "" {
				return flag.ErrHelp
			}

			signer, err := GetSigner(ctx, *sk, *key)
			if err != nil {
				return err
			}

			return KeyPOPSignCmd(ctx, *challenge, *nonce, signer)
		},
	}
}

func KeyPOPSignCmd(ctx context.Context,
	challenge, nonce string,
	signer signature.Signer) error {
	sig, err := DoKeyPOPSign(ctx, challenge, nonce, signer)

	if err != nil {
		return err
	}
	fmt.Println(base64.StdEncoding.EncodeToString(sig))

	return nil
}

func DoKeyPOPSign(ctx context.Context,
	challenge, nonce string,
	signer signature.Signer) ([]byte, error) {
	data := PAE(challenge, nonce)
	return signer.SignMessage(bytes.NewReader(data),
		options.WithContext(ctx))

}
