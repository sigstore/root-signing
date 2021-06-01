package keys

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
)

// See https://developers.yubico.com/PIV/Introduction/PIV_attestation.html
var oidExtensionSerialNumber = []int{1, 3, 6, 1, 4, 1, 41482, 3, 7}

// SigningKey contains the serial number, public key, device cert, and key cert.
type SigningKey struct {
	SerialNumber int
	PublicKey    *ecdsa.PublicKey
	DeviceCert   *x509.Certificate
	KeyCert      *x509.Certificate
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

func getSerialNumber(c *x509.Certificate) (*int, error) {
	// Retrieves the serial number from the OID extension in the certificate
	for _, e := range c.Extensions {
		if e.Id.Equal(oidExtensionSerialNumber) {
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
