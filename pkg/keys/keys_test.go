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
	"os"
	"testing"

	csignature "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/theupdateframework/go-tuf/pkg/keys"
)

// Generated with:
// openssl genpkey -algorithm ED25519 -out edprivate.pem
const ed25519PublicKey = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIKjlXfR/VFvO9qM9+CG2qbuSM54k8ciKWHhgNwKTgqpG
-----END PRIVATE KEY-----
`

// Generated with cosign piv-tool
const ecdsaPublicKey = `
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEMsLvdEM1SnWcdXXNa5NcwsrG7Mpf
D1ujmb0yPLLykhzpi1GzEiSYT4BfBB3GX08G3+rWxZAi8Ilhu62L8s4JpA==
-----END PUBLIC KEY-----
`

// Generated with cosign piv-tool
const keyCert = `-----BEGIN CERTIFICATE-----
MIICRDCCASygAwIBAgIQadj3MkImEj+VDi7jru0/5TANBgkqhkiG9w0BAQsFADAh
MR8wHQYDVQQDDBZZdWJpY28gUElWIEF0dGVzdGF0aW9uMCAXDTE2MDMxNDAwMDAw
MFoYDzIwNTIwNDE3MDAwMDAwWjAlMSMwIQYDVQQDDBpZdWJpS2V5IFBJViBBdHRl
c3RhdGlvbiA5YzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABDLC73RDNUp1nHV1
zWuTXMLKxuzKXw9bo5m9Mjyy8pIc6YtRsxIkmE+AXwQdxl9PBt/q1sWQIvCJYbut
i/LOCaSjPTA7MBEGCisGAQQBgsQKAwMEAwQEBTAUBgorBgEEAYLECgMHBAYCBACg
/EUwEAYKKwYBBAGCxAoDCAQCAwIwDQYJKoZIhvcNAQELBQADggEBAD0pDMAg6LME
AW3vPN//0beH7EP+yCIgeXLBAcMnUK02XhoXHs5vGQzcrfgl2izcV7QcZznYv1Ou
sladMoIUOhuNojPZWNCP301EpiNFTMywpndxjSyIrtVPCwOM6yqIc8A0lrRkTyse
hkEYQqGeJa1Vz5VGid/7fjUSaTaZDWT1oNqSNjnV0SkUr/nA6q9RJ8WCBu8adIfz
FMI8CX/DV8OpF5SlLXXzLcfyNL4dyRrrpH5zS665JQT72ZWA3yuLP1R3o9cWiyZi
eXccKEXYp90X5WxWKam6mwkNrgoOWaTDUPpeveMJwHu2D+e38U3F6KLJhXsREuXP
y7FIbzkFc0g=
-----END CERTIFICATE-----`

// Generated with cosign piv-tool
const deviceCert = `-----BEGIN CERTIFICATE-----
MIIDDTCCAfWgAwIBAgIJAMHMibcEuZYWMA0GCSqGSIb3DQEBCwUAMCsxKTAnBgNV
BAMMIFl1YmljbyBQSVYgUm9vdCBDQSBTZXJpYWwgMjYzNzUxMCAXDTE2MDMxNDAw
MDAwMFoYDzIwNTIwNDE3MDAwMDAwWjAhMR8wHQYDVQQDDBZZdWJpY28gUElWIEF0
dGVzdGF0aW9uMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA43jxRyx5
M5h7uTFmU/MKus77xCT50usFB9NuWa7RrCdEPWSU8+zrUmfwxphdDgarwVD6lvWn
FRUBpRvcnX26copHHWHe9iprAoGCL6iqmqXXcz49Xg9DmcxNlUtomlbCQRYZzHEa
k3W2vUE9Tci00e4q3rxWZZD/S5CuCLssJMXYxFwERedIZUhDmtMk46RP3R6qn4/Z
lF53Ck2IIfuNqb3SNAiTWmwNYtyZt3V5xIvZAjyMfkcvNJW4F19SsGHb+dnwhLBA
dXyUzl3brJN1XFHaGFAfmgBKTh2Cibz622fTj00ICezOEMnh67+1jfEr8EbuLTzF
L6fkCZMZQ3iVNQIDAQABozwwOjARBgorBgEEAYLECgMDBAMEBAUwEQYKKwYBBAGC
xAoDCgQDAgEDMBIGA1UdEwEB/wQIMAYBAf8CAQAwDQYJKoZIhvcNAQELBQADggEB
AKuBRRECT6KrYH1/vjVpCP1A1JdIU0zM5DWhQ5lXaXFXknYK+OAfrwCGs/c0yPXU
jfjXlcpPZq1jWjzLTP5MEDJ/RCoZPNB9UH4Zh5KfqKPlBZ9VQ0eFjGmA3ny1vLFk
RljMj2nctsUaOHXBrD2c2xBSN0/Jwo8IQRnCBNG4ZTcrvIkkx2LZ5xxTkX1r6c8V
UzuhD3NM97M8WzT/PmZOwRSK8iiWDRgD2VxWddg4RlL32gsE+/L9+j3sr0jhzKQf
62DGzb04kO2+4zqMVNH83Ho+9PnvtUPC7VTId2UBc8D1JBZCN7gBwRp934NfQlBP
gUPpyzra1/D3eME/ixhdtcw=
-----END CERTIFICATE-----`

const rootCA = `-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIDBAZHMA0GCSqGSIb3DQEBCwUAMCsxKTAnBgNVBAMMIFl1
YmljbyBQSVYgUm9vdCBDQSBTZXJpYWwgMjYzNzUxMCAXDTE2MDMxNDAwMDAwMFoY
DzIwNTIwNDE3MDAwMDAwWjArMSkwJwYDVQQDDCBZdWJpY28gUElWIFJvb3QgQ0Eg
U2VyaWFsIDI2Mzc1MTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMN2
cMTNR6YCdcTFRxuPy31PabRn5m6pJ+nSE0HRWpoaM8fc8wHC+Tmb98jmNvhWNE2E
ilU85uYKfEFP9d6Q2GmytqBnxZsAa3KqZiCCx2LwQ4iYEOb1llgotVr/whEpdVOq
joU0P5e1j1y7OfwOvky/+AXIN/9Xp0VFlYRk2tQ9GcdYKDmqU+db9iKwpAzid4oH
BVLIhmD3pvkWaRA2H3DA9t7H/HNq5v3OiO1jyLZeKqZoMbPObrxqDg+9fOdShzgf
wCqgT3XVmTeiwvBSTctyi9mHQfYd2DwkaqxRnLbNVyK9zl+DzjSGp9IhVPiVtGet
X02dxhQnGS7K6BO0Qe8CAwEAAaNCMEAwHQYDVR0OBBYEFMpfyvLEojGc6SJf8ez0
1d8Cv4O/MA8GA1UdEwQIMAYBAf8CAQEwDgYDVR0PAQH/BAQDAgEGMA0GCSqGSIb3
DQEBCwUAA4IBAQBc7Ih8Bc1fkC+FyN1fhjWioBCMr3vjneh7MLbA6kSoyWF70N3s
XhbXvT4eRh0hvxqvMZNjPU/VlRn6gLVtoEikDLrYFXN6Hh6Wmyy1GTnspnOvMvz2
lLKuym9KYdYLDgnj3BeAvzIhVzzYSeU77/Cupofj093OuAswW0jYvXsGTyix6B3d
bW5yWvyS9zNXaqGaUmP3U9/b6DlHdDogMLu3VLpBB9bm5bjaKWWJYgWltCVgUbFq
Fqyi4+JE014cSgR57Jcu3dZiehB6UtAPgad9L5cNvua/IWRmm+ANy3O2LH++Pyl8
SREzU8onbBsjMg9QDiSf5oJLKvd/Ren+zGY7
-----END CERTIFICATE-----`

func TestToSigningKey(t *testing.T) {
	tests := []struct {
		name          string
		pub           []byte
		deviceCert    []byte
		keyCert       []byte
		expectSuccess bool
	}{
		{
			name:          "valid yubikey",
			pub:           []byte(ecdsaPublicKey),
			deviceCert:    []byte(deviceCert),
			keyCert:       []byte(keyCert),
			expectSuccess: true,
		},
		{
			name:          "missing key",
			deviceCert:    []byte(deviceCert),
			keyCert:       []byte(keyCert),
			expectSuccess: false,
		},
		{
			name:          "missing key cert",
			pub:           []byte(ecdsaPublicKey),
			deviceCert:    []byte(deviceCert),
			expectSuccess: false,
		},
		{
			name:          "missing device cert",
			pub:           []byte(ecdsaPublicKey),
			keyCert:       []byte(keyCert),
			expectSuccess: false,
		},
		{
			name:          "invalid key (ed25519)",
			pub:           []byte(ed25519PublicKey),
			deviceCert:    []byte(deviceCert),
			keyCert:       []byte(keyCert),
			expectSuccess: false,
		},
		{
			name:          "invalid key cert",
			pub:           []byte(ecdsaPublicKey),
			deviceCert:    []byte(deviceCert),
			keyCert:       []byte("abc"),
			expectSuccess: false,
		},
		{
			name:          "invalid device cert",
			pub:           []byte(ecdsaPublicKey),
			deviceCert:    []byte("abc"),
			keyCert:       []byte(keyCert),
			expectSuccess: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ToSigningKey(123, tt.pub, tt.deviceCert, tt.keyCert)
			if tt.expectSuccess != (err == nil) {
				t.Errorf("unexpected error generating signing key (%s): %s", tt.name, err)
			}
			if tt.expectSuccess {
				hexPubKey, err := EcdsaTufKey(key.PublicKey, true)
				if err != nil {
					t.Errorf("unexpected error generating hex TUF public key: %s", err)
				}
				pemPubKey, err := EcdsaTufKey(key.PublicKey, false)
				if err != nil {
					t.Errorf("unexpected error generating PEM TUF public key: %s", err)
				}
				// True to get verifiers.
				_, err = keys.GetVerifier(hexPubKey)
				if err != nil {
					t.Errorf("unexpected error getting TUF hex ecdsa verifier: %s", err)
				}
				_, err = keys.GetVerifier(pemPubKey)
				if err != nil {
					t.Errorf("unexpected error getting TUF PEM ecdsa verifier: %s", err)
				}
			}
		})
	}
}

func TestGetSigningKey(t *testing.T) {
	ctx := context.Background()

	keyRef := "../../tests/test_data/cosign.key"
	signingKey, err := csignature.SignerVerifierFromKeyRef(ctx, keyRef, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid signing key with PEM", func(t *testing.T) {
		signingKeyPem, err := ConstructTufKey(ctx, signingKey, false)
		if err != nil {
			t.Fatal(err)
		}
		_, err = keys.GetVerifier(signingKeyPem)
		if err != nil {
			t.Fatalf("unexpected error getting TUF PEM ecdsa verifier: %s", err)
		}
		// Try with explicit verifier.
		pemKey := keys.NewEcdsaVerifier()
		if err := pemKey.UnmarshalPublicKey(signingKeyPem); err != nil {
			t.Errorf("unexpected error getting TUF PEM ecdsa verifier: %s", err)
		}
	})
	t.Run("valid signing key with hex", func(t *testing.T) {
		signingKeyHex, err := ConstructTufKey(ctx, signingKey, true)
		if err != nil {
			t.Fatal(err)
		}
		_, err = keys.GetVerifier(signingKeyHex)
		if err != nil {
			t.Fatalf("unexpected error getting hex PEM ecdsa verifier: %s", err)
		}
		// Try with explicit verifier.
		hexKey := keys.NewDeprecatedEcdsaVerifier()
		if err := hexKey.UnmarshalPublicKey(signingKeyHex); err != nil {
			t.Errorf("unexpected error unmarshalling with  TUF hex ecdsa verifier: %s", err)
		}
		// Fails with other verifier.
		pemKey := keys.NewEcdsaVerifier()
		if err := pemKey.UnmarshalPublicKey(signingKeyHex); err == nil {
			t.Errorf("expected error unmarshalling with TUF PEM ecdsa verifier: %s", err)
		}
	})
}

func TestVerify(t *testing.T) {
	pubkey, err := os.ReadFile("../../tests/test_data/10550341/10550341_pubkey.pem")
	if err != nil {
		t.Fatal("error opening test data")
	}
	deviceCert, err := os.ReadFile("../../tests/test_data/10550341/10550341_device_cert.pem")
	if err != nil {
		t.Fatal("error opening test data")
	}
	keyCert, err := os.ReadFile("../../tests/test_data/10550341/10550341_key_cert.pem")
	if err != nil {
		t.Fatal("error opening test data")
	}

	signingKey, err := ToSigningKey(10550341, pubkey, deviceCert, keyCert)
	if err != nil {
		t.Fatal("error creating signing key")
	}

	// Use Yubico root CA
	roots, err := cryptoutils.UnmarshalCertificatesFromPEMLimited([]byte(rootCA), 1)
	if err != nil {
		t.Fatal("error creating root CA certificate")
	}

	{
		// Verify signing key
		if err = signingKey.Verify(roots[0]); err != nil {
			t.Fatal("unexpected error verifying signing key")
		}
	}

	// Remove serial number from cert, expect failure.
	{
		badSigningKey := signingKey
		badSigningKey.KeyCert.Extensions = nil
		if err := badSigningKey.Verify(roots[0]); err == nil {
			t.Fatal("expected error verifying signing key with missing serial number in key cert")
		}
	}

	// Change serial number, expect failure.
	{
		badSigningKey := signingKey
		badSigningKey.SerialNumber = 123
		if err := badSigningKey.Verify(roots[0]); err == nil {
			t.Fatal("expected error verifying signing key with mismatching serial number")
		}
	}
}
