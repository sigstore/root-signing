package main

import (
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asraa/sigstore-root/scripts/pkg/keys"
)

type file string

func (f *file) String() string {
	return string(*f)
}

func (f *file) Set(s string) error {
	if s == "" {
		return errors.New("flag must be specified")
	}
	if _, err := os.Stat(filepath.Clean(s)); os.IsNotExist(err) {
		return err
	}
	*f = file(s)
	return nil
}

func toCert(filename string) (*x509.Certificate, error) {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return keys.ToCert(fileBytes)
}

func signingKeyFromDir(dirname string) (*keys.SigningKey, error) {
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
			pubKey, err = ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(info.Name(), "_key_cert.pem") {
			keyCert, err = ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(info.Name(), "_device_cert.pem") {
			deviceCert, err = ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys.ToSigningKey(serial, pubKey, deviceCert, keyCert)
}

func verifySigningKeys(dirname string, rootCA *x509.Certificate) error {
	// Get all signing keys in the directory.
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			key, err := signingKeyFromDir(filepath.Join(dirname, file.Name()))
			if err != nil {
				return err
			}
			if err = key.Verify(rootCA); err != nil {
				log.Printf("error verifying key %d: %s", key.SerialNumber, err)
				return err
			} else {
				log.Printf("verified key %d", key.SerialNumber)
			}
		}
	}
	return nil
}

func main() {
	var fileFlag file
	flag.Var(&fileFlag, "root", "Yubico root certificate")
	keyDir := flag.String("key-directory", "../../../ceremony/2021-05-03/keys", "Directory with key products")
	// TODO: Add path to repository to verify metadata.
	flag.Parse()

	rootCA, err := toCert(string(fileFlag))
	if err != nil {
		log.Printf("failed to parse root CA: %s", err)
		os.Exit(1)
	}

	if err := verifySigningKeys(*keyDir, rootCA); err != nil {
		log.Printf("error verifying signing keys: %s", err)
		os.Exit(1)
	}
}
