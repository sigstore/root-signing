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
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/options"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/pkg/keys"

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

type KeyValue struct {
	PublicKey string `json:"public"`
}

func (kv *KeyValue) Unmarshal(pubKey *data.PublicKey) error {
	return json.Unmarshal(pubKey.Value, kv)
}

func ToCert(pemBytes []byte) (*x509.Certificate, error) {
	certs, err := cryptoutils.UnmarshalCertificatesFromPEMLimited(pemBytes, 2)
	if err != nil {
		return nil, err
	}
	if len(certs) != 1 {
		return nil, errors.New("expected one PEM encoded certificate")
	}
	return certs[0], nil
}

func ToSigningKey(serialNumber int, pubKey []byte, deviceCert []byte, keyCert []byte) (*SigningKey, error) {
	// Creates a signing key from the PEM bytes of the public key, device cert, and key cert
	var err error
	pk, err := cryptoutils.UnmarshalPEMToPublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	ecdsapk, ok := pk.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("expected ecdsa public key")
	}
	key := &SigningKey{
		SerialNumber: serialNumber,
		PublicKey:    ecdsapk}
	key.DeviceCert, err = ToCert(deviceCert)
	if err != nil {
		return nil, err
	}
	key.KeyCert, err = ToCert(keyCert)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// EcdsaTufKey returns a PEM-encoded TUF public key for an ecdsa key.
func EcdsaTufKey(pub *ecdsa.PublicKey) (*data.PublicKey, error) {
	keyValBytes, err := json.Marshal(keys.EcdsaVerifier{
		PublicKey: &keys.PKIXPublicKey{PublicKey: pub}})
	if err != nil {
		return nil, err
	}
	return &data.PublicKey{
		// TODO: Update to new format for next key signing
		Type:       data.KeyTypeECDSA_SHA2_P256_OLD_FMT,
		Scheme:     data.KeySchemeECDSA_SHA2_P256,
		Algorithms: data.HashAlgorithms,
		Value:      keyValBytes,
	}, nil
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
		switch {
		case strings.HasSuffix(info.Name(), "_pubkey.pem"):
			pubKey, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		case strings.HasSuffix(info.Name(), "_key_cert.pem"):
			keyCert, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		case strings.HasSuffix(info.Name(), "_device_cert.pem"):
			deviceCert, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		default:
			return nil
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

// ConstructTufKey constructs a TUF public key from a given signer.
func ConstructTufKey(ctx context.Context, signer signature.Signer) (*data.PublicKey, error) {
	pub, err := signer.PublicKey(options.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return ConstructTufKeyFromPublic(ctx, pub)
}

// ConstructTufKey constructs a TUF public key from a public key
func ConstructTufKeyFromPublic(_ context.Context, pubKey crypto.PublicKey) (*data.PublicKey, error) {
	switch kt := pubKey.(type) {
	case *ecdsa.PublicKey:
		return EcdsaTufKey(kt)
	default:
		return nil, fmt.Errorf("ConstructTufKeyFromPublic: key type %s not supported", kt)
	}
}
