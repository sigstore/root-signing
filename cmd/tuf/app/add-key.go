//go:build pivkey
// +build pivkey

package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/sigstore/cosign/cmd/cosign/cli/pivcli"
	"github.com/theupdateframework/go-tuf/data"
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
	attestations pivcli.Attestations
	key          *data.Key
}

func GetKeyAndAttestation(ctx context.Context) (*KeyAndAttestations, error) {
	attestations, err := pivcli.AttestationCmd(ctx, "signature")
	if err != nil {
		return nil, err
	}

	pub := attestations.KeyCert.PublicKey.(*ecdsa.PublicKey)
	keyValBytes, err := json.Marshal(ecdsaPublic{PublicKey: elliptic.Marshal(pub.Curve, pub.X, pub.Y)})
	if err != nil {
		return nil, err
	}
	pk := &data.PublicKey{
		Type:       data.KeyTypeECDSA_SHA2_P256,
		Scheme:     data.KeySchemeECDSA_SHA2_P256,
		Algorithms: data.HashAlgorithms,
		Value:      keyValBytes,
	}

	return &KeyAndAttestations{attestations: *attestations, key: pk}, nil
}

func AddKeyCmd(ctx context.Context, directory string) error {
	if err := pivcli.ResetKeyCmd(ctx); err != nil {
		return err
	}

	if err := pivcli.GenerateKeyCmd(ctx, "", true, "signature", "always", "always"); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Resetting PIN. Enter a new PIN between 6 and 8 characters: ")
	pin, err := term.ReadPassword(0)
	if err != nil {
		return err
	}
	fmt.Println(os.Stderr)
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

type ecdsaPublic struct {
	PublicKey data.HexBytes `json:"public"`
}

func WriteKeyData(keyAndAttestations *KeyAndAttestations, directory string) error {
	att := keyAndAttestations.attestations
	serial := fmt.Sprint(att.KeyAttestation.Serial)
	keyDir := filepath.Join(directory, "keys", serial)
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return err
	}

	b, err := x509.MarshalPKIXPublicKey(keyAndAttestations.attestations.KeyCert.PublicKey)
	if err != nil {
		return err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: b,
	})
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
