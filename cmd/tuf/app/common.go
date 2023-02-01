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
	"context"
	"crypto"
	"fmt"

	"github.com/sigstore/cosign/pkg/cosign/pivkey"
	csignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
)

func GetSigner(ctx context.Context, sk bool, keyRef string) (signature.Signer, error) {
	if sk {
		//nolint: staticcheck
		pivKey, err := pivkey.GetKeyWithSlot("signature")
		//nolint: staticcheck
		if err != nil {
			return nil, err
		}
		return pivKey.SignerVerifier()
	}
	// A key reference was provided.
	// First try to load it as a regular PEM encoded private key.
	signer, err := signature.LoadSignerFromPEMFile(keyRef, crypto.SHA256, nil)
	if err != nil {
		var innerError error
		signer, innerError = csignature.SignerVerifierFromKeyRef(ctx, keyRef, nil)
		if innerError != nil {
			// Only print this message if both attempts failed.
			// As there is a natual fallthrough here, always
			// logging the first error could be noisy.
			fmt.Printf("failed to load key as PEM encoded: %s, trying other methods: ", err)
			return nil, innerError
		}

	}
	return signer, nil
}

func GetVerifier(ctx context.Context, keyRef string) (signature.Verifier, error) {
	verifier, err := signature.LoadVerifierFromPEMFile(keyRef, crypto.SHA256)

	return verifier, err
}

// The DSSE Pre-Authentication Encoding
// https://github.com/secure-systems-lab/dsse/blob/master/protocol.md#signature-definition
func PAE(challenge, nonce string) []byte {
	return []byte(fmt.Sprintf("key-kop-v1 %d %s %d %s",
		len(challenge), challenge,
		len(nonce), nonce))
}
