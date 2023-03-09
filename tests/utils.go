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
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-piv/piv-go/piv"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/pivcli"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	csignature "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/root-signing/cmd/tuf/app"
	"github.com/sigstore/root-signing/pkg/keys"
	"github.com/sigstore/root-signing/pkg/repo"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf"
)

type repoTestStack struct {
	ctx           context.Context
	repoDir       string
	hsmRootCA     *x509.Certificate
	hsmRootSigner crypto.PrivateKey
	targetsConfig *repo.TargetMetaConfig
	snapshotRef   string
	timestampRef  string
}

// newRepoTestStack initializes a test stack for e2e
// testing, setting up a root CA for the HSM keys
func newRepoTestStack(ctx context.Context, t *testing.T) *repoTestStack {
	td := t.TempDir()

	rootCA, rootSigner, err := CreateRootCA()
	if err != nil {
		t.Fatal(err)
	}

	return &repoTestStack{
		repoDir:       td,
		hsmRootCA:     rootCA,
		hsmRootSigner: rootSigner,
		targetsConfig: &repo.TargetMetaConfig{
			Add: map[string]json.RawMessage{},
			Del: map[string]json.RawMessage{},
		},
		snapshotRef:  createTestSigner(t),
		timestampRef: createTestSigner(t),
	}
}

func (s *repoTestStack) addTarget(t *testing.T, name, content string, custom json.RawMessage) {
	testTarget := filepath.Join(s.repoDir, name)
	if err := os.WriteFile(testTarget, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	s.targetsConfig.Add[name] = custom
}

func (s *repoTestStack) removeTarget(t *testing.T, name string) {
	delete(s.targetsConfig.Add, name)
}

func (s *repoTestStack) genKey(t *testing.T, hsm bool) string {
	if hsm {
		keyRef, err := CreateTestHsmSigner(s.repoDir, s.hsmRootCA, s.hsmRootSigner)
		if err != nil {
			t.Fatal(err)
		}
		return keyRef
	}
	return createTestSigner(t)
}

func (s *repoTestStack) getSigner(t *testing.T, ref string) signature.Signer {
	signer, err := signature.LoadSignerFromPEMFile(ref, crypto.SHA256, nil)
	if err != nil {
		signer, err := csignature.SignerVerifierFromKeyRef(s.ctx, ref, nil)
		if err != nil {
			t.Fatal(err)
		}
		return signer
	}
	return signer
}

func (s *repoTestStack) removeHsmKey(t *testing.T, ref string) {
	if err := os.RemoveAll(filepath.Dir(ref)); err != nil {
		t.Fatal(err)
	}
}

func (s *repoTestStack) getManifest(t *testing.T, manifest string) json.RawMessage {
	store := tuf.FileSystemStore(s.repoDir, nil)
	meta, err := store.GetMeta()
	if err != nil {
		t.Fatal(err)
	}
	md, ok := meta[manifest]
	if !ok {
		t.Fatalf("missing %s", manifest)
	}
	return md
}

func (s *repoTestStack) snapshot(t *testing.T) {
	if err := app.SnapshotCmd(s.ctx, s.repoDir); err != nil {
		t.Fatalf("expected Snapshot command to pass, got err: %s", err)
	}
	snapshotSigner, err := csignature.SignerVerifierFromKeyRef(s.ctx, s.snapshotRef, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(s.ctx, s.repoDir, []string{"snapshot"}, snapshotSigner, false); err != nil {
		t.Fatal(err)
	}
}

func (s *repoTestStack) timestamp(t *testing.T) {
	if err := app.TimestampCmd(s.ctx, s.repoDir); err != nil {
		t.Fatalf("expected timestamp command to pass, got err: %s", err)
	}
	timestampSigner, err := csignature.SignerVerifierFromKeyRef(s.ctx, s.timestampRef, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SignCmd(s.ctx, s.repoDir, []string{"timestamp"}, timestampSigner, false); err != nil {
		t.Fatal(err)
	}
}

func (s *repoTestStack) publish(t *testing.T) {
	if err := app.PublishCmd(s.ctx, s.repoDir); err != nil {
		t.Fatal(err)
	}
}

// Create fake key signer in testDirectory. Returns file reference to signer.
func createTestSigner(t *testing.T) string {
	keys, err := cosign.GenerateKeyPair(nil)
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	f, _ := os.CreateTemp(temp, "*.key")

	if _, err := io.Copy(f, bytes.NewBuffer(keys.PrivateBytes)); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

// Create fake key signer in testDirectory. Returns file reference to signer
// and verifier.
func createTestSignVerifier(t *testing.T) (string, string) {
	keys, err := cosign.GenerateKeyPair(nil)
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	priv, _ := os.CreateTemp(temp, "*.key")
	pub, _ := os.CreateTemp(temp, "*.pub")

	if _, err := io.Copy(priv, bytes.NewBuffer(keys.PrivateBytes)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(pub, bytes.NewBuffer(keys.PublicBytes)); err != nil {
		t.Fatal(err)
	}

	return priv.Name(), pub.Name()
}

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

func CreateTestHsmSigner(testDir string, root *x509.Certificate, rootSigner crypto.PrivateKey) (string, error) {
	generated, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}
	pub := &generated.PublicKey
	n, _ := rand.Int(rand.Reader, big.NewInt(100000000))
	serial := uint32(n.Uint64())

	deviceCert, keyCert, err := createTestAttestations(root, rootSigner, pub, int(serial))
	if err != nil {
		return "", err
	}

	deviceCertPem, err := cryptoutils.MarshalCertificateToPEM(deviceCert)
	if err != nil {
		return "", err
	}
	keyCertPem, err := cryptoutils.MarshalCertificateToPEM(keyCert)
	if err != nil {
		return "", err
	}

	keyAndAttestations := &app.KeyAndAttestations{
		Key: pub,
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
		return "", err
	}

	b, err := cryptoutils.MarshalPrivateKeyToPEM(generated)
	if err != nil {
		return "", err
	}

	serialStr := fmt.Sprint(keyAndAttestations.Attestations.KeyAttestation.Serial)
	privKeyFile := filepath.Join(testDir, "keys", serialStr, serialStr+"_privkey.pem")
	if err := ioutil.WriteFile(privKeyFile, []byte(b), 0644); err != nil {
		return "", err
	}
	return privKeyFile, nil
}
