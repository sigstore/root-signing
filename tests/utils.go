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

package test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"time"

	"github.com/go-piv/piv-go/piv"
	"github.com/sigstore/cosign/cmd/cosign/cli/pivcli"
	"github.com/sigstore/root-signing/cmd/tuf/app"
	"github.com/sigstore/root-signing/pkg/keys"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
)

func CreateRootCA() (*x509.Certificate, crypto.PrivateKey, error) {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2022),
		Subject: pkix.Name{
			Organization: []string{"Test Root."},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	cert, _ := x509.ParseCertificate(caBytes)

	return cert, caPrivKey, nil
}

// Returns a test device cert and key cert.
func createTestAttestations(root *x509.Certificate, rootSigner crypto.PrivateKey, pk crypto.PublicKey, serial int) (*x509.Certificate, *x509.Certificate, error) {
	// create device cert
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2022),
		Subject: pkix.Name{
			Organization: []string{"Test Device Cert"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	devicePrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// create the CA
	deviceBytes, err := x509.CreateCertificate(rand.Reader, ca, root, &devicePrivKey.PublicKey, rootSigner)
	if err != nil {
		return nil, nil, err
	}
	deviceCert, _ := x509.ParseCertificate(deviceBytes)

	// KeyCert must contain serial number as OID extension.
	serialized, _ := asn1.Marshal(serial)
	keyCertTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2022),
		Subject: pkix.Name{
			Organization: []string{"Test Device Cert"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		ExtraExtensions:       []pkix.Extension{{Id: keys.OidExtensionSerialNumber, Value: serialized}},
	}
	// create the cert
	keyCertBytes, err := x509.CreateCertificate(rand.Reader, keyCertTemplate, deviceCert, pk, devicePrivKey)
	if err != nil {
		return nil, nil, err
	}
	keyCert, _ := x509.ParseCertificate(keyCertBytes)

	return deviceCert, keyCert, nil
}

func GetTestHsmSigner(ctx context.Context, testDir string, serial uint32) (*keys.SignerAndTufKey, error) {
	// read private key from file.
	serialStr := fmt.Sprint(serial)
	privKeyFile := filepath.Join(testDir, "keys", serialStr, serialStr+"_privkey.pem")

	signer, err := signature.LoadSignerVerifierFromPEMFile(privKeyFile, crypto.SHA256, nil)
	if err != nil {
		return nil, err
	}

	cryptoPub, _ := signer.PublicKey()
	pub := cryptoPub.(*ecdsa.PublicKey)

	pk, err := keys.EcdsaTufKey(pub)
	if err != nil {
		return nil, err
	}

	return &keys.SignerAndTufKey{Signer: signer, Key: pk}, nil
}

func CreateTestHsmSigner(testDir string, root *x509.Certificate, rootSigner crypto.PrivateKey) (*uint32, error) {
	generated, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := &generated.PublicKey
	n, _ := rand.Int(rand.Reader, big.NewInt(100000000))
	serial := uint32(n.Uint64())

	deviceCert, keyCert, err := createTestAttestations(root, rootSigner, pub, int(serial))
	if err != nil {
		return nil, err
	}

	pk, err := keys.EcdsaTufKey(pub)
	if err != nil {
		return nil, err
	}

	deviceCertPem, err := cryptoutils.MarshalCertificateToPEM(deviceCert)
	if err != nil {
		return nil, err
	}
	keyCertPem, err := cryptoutils.MarshalCertificateToPEM(keyCert)
	if err != nil {
		return nil, err
	}

	keyAndAttestations := &app.KeyAndAttestations{
		Key: pk,
		Attestations: pivcli.Attestations{
			DeviceCert:    deviceCert,
			DeviceCertPem: string(deviceCertPem),
			KeyCert:       keyCert,
			KeyCertPem:    string(keyCertPem),
			KeyAttestation: &piv.Attestation{
				Serial: serial,
			},
		},
	}

	// Write to repository/keys/SERIAL_NUM/SERIAL_NUM_pubkey.pem, etc
	if err := app.WriteKeyData(keyAndAttestations, testDir); err != nil {
		return nil, err
	}

	b, err := cryptoutils.MarshalPrivateKeyToPEM(generated)
	if err != nil {
		return nil, err
	}

	serialStr := fmt.Sprint(keyAndAttestations.Attestations.KeyAttestation.Serial)
	privKeyFile := filepath.Join(testDir, "keys", serialStr, serialStr+"_privkey.pem")
	if err := ioutil.WriteFile(privKeyFile, []byte(b), 0644); err != nil {
		return nil, err
	}
	return &serial, nil
}
