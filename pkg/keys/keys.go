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

package keys

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	csignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/options"
	"github.com/theupdateframework/go-tuf/data"

	// Register the provider-specific plugins
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
)

// See https://developers.yubico.com/PIV/Introduction/PIV_attestation.html
var OidExtensionSerialNumber = []int{1, 3, 6, 1, 4, 1, 41482, 3, 7}

// SigningKey contains the serial number, public key, device cert, and key cert.
type SigningKey struct {
	SerialNumber int
	PublicKey    *ecdsa.PublicKey
	DeviceCert   *x509.Certificate
	KeyCert      *x509.Certificate
}

type SignerAndTufKey struct {
	Signer signature.Signer
	Key    *data.PublicKey
}

type EcdsaPublic struct {
	PublicKey data.HexBytes `json:"public"`
}

func toPubKey(pemBytes []byte) (*ecdsa.PublicKey, error) {
	if pemBytes == nil {
		return nil, errors.New("failed to get public key")
	}
	block, rest := pem.Decode(pemBytes)
	if len(rest) != 0 {
		return nil, errors.New("failed to parse public key PEM block")
	}
	pubkey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pubkey.(*ecdsa.PublicKey), nil
}

func ToCert(pemBytes []byte) (*x509.Certificate, error) {
	if pemBytes == nil {
		return nil, errors.New("failed to get cert")
	}
	block, rest := pem.Decode(pemBytes)
	if len(rest) != 0 {
		return nil, errors.New("failed to parse certificate PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func ToSigningKey(serialNumber int, pubKey []byte, deviceCert []byte, keyCert []byte) (*SigningKey, error) {
	// Creates a signing key from the PEM bytes of the public key, device cert, and key cert
	key := SigningKey{SerialNumber: serialNumber}
	var err error
	if key.PublicKey, err = toPubKey(pubKey); err != nil {
		return nil, err
	}
	if key.DeviceCert, err = ToCert(deviceCert); err != nil {
		return nil, err
	}
	if key.KeyCert, err = ToCert(keyCert); err != nil {
		return nil, err
	}
	return &key, nil
}

func EcdsaTufKey(pub *ecdsa.PublicKey) (*data.PublicKey, error) {
	keyValBytes, err := json.Marshal(EcdsaPublic{PublicKey: elliptic.Marshal(pub.Curve, pub.X, pub.Y)})
	if err != nil {
		return nil, err
	}
	return &data.PublicKey{
		Type:       data.KeyTypeECDSA_SHA2_P256,
		Scheme:     data.KeySchemeECDSA_SHA2_P256,
		Algorithms: data.HashAlgorithms,
		Value:      keyValBytes,
	}, nil
}

func ToTufKey(key SigningKey) (*data.PublicKey, error) {
	return EcdsaTufKey(key.PublicKey)
}

func getSerialNumber(c *x509.Certificate) (*int, error) {
	// Retrieves the serial number from the OID extension in the certificate
	for _, e := range c.Extensions {
		if e.Id.Equal(OidExtensionSerialNumber) {
			var serial int
			if rest, err := asn1.Unmarshal(e.Value, &serial); err != nil {
				return nil, err
			} else if len(rest) != 0 {
				return nil, errors.New("error unmarshalling serial number")
			}
			return &serial, nil
		}
	}
	return nil, errors.New("missing serial number in certificate")
}

func SigningKeyFromDir(dirname string) (*SigningKey, error) {
	// Expect *_device_cert.pem, *_key_cert.pem, *_pubkey.pem in each key directory.
	serialStr := filepath.Base(dirname)
	serial, err := strconv.Atoi(serialStr)
	if err != nil {
		return nil, fmt.Errorf("invalid key directory name %s: %s", dirname, err)
	}

	var pubKey []byte
	var deviceCert []byte
	var keyCert []byte
	err = filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("panic accessing path %q: %v\n", path, err)
			return err
		}
		if strings.HasSuffix(info.Name(), "_pubkey.pem") {
			pubKey, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(info.Name(), "_key_cert.pem") {
			keyCert, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(info.Name(), "_device_cert.pem") {
			deviceCert, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ToSigningKey(serial, pubKey, deviceCert, keyCert)
}

func (key SigningKey) Verify(root *x509.Certificate) error {
	// Verify against root.
	roots := x509.NewCertPool()
	roots.AddCert(root)
	intermediates := x509.NewCertPool()
	intermediates.AddCert(key.DeviceCert)

	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
	}

	// Verify the chain from key cert to root CA with intermediate device cert
	if _, err := key.KeyCert.Verify(opts); err != nil {
		return err
	}

	// Verify dirname matches serial number from the key cert extension
	serialNumber, err := getSerialNumber(key.KeyCert)
	if err != nil {
		return fmt.Errorf("error getting serial number from cert: %s", err)
	}
	if key.SerialNumber != *serialNumber {
		return fmt.Errorf("serial number does not match certificate for key expected %d, got %d", key.SerialNumber, *serialNumber)
	}
	return nil
}

func GetSigningKey(ctx context.Context, keyRef string) (*SignerAndTufKey, error) {
	key, err := csignature.SignerVerifierFromKeyRef(ctx, keyRef, nil)
	if err != nil {
		return nil, err
	}
	pub, err := key.PublicKey(options.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	switch kt := pub.(type) {
	case *ecdsa.PublicKey:
		pk, err := EcdsaTufKey(kt)
		if err != nil {
			return nil, err
		}
		return &SignerAndTufKey{
			Key:    pk,
			Signer: key}, nil
	case *rsa.PublicKey:
		return nil, errors.New("RSA keys not supported")

	}
	return nil, errors.New("not an ecdsa or rsa key")
}
