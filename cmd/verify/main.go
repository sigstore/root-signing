package main

import (
	"crypto/x509"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/sigstore/root-signing/pkg/keys"
	"github.com/sigstore/root-signing/pkg/repo"
	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/verify"
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

// Map from Key ID to Signing Key
type KeyMap map[string]*keys.SigningKey

func getKeyID(key keys.SigningKey) (*string, error) {
	pk := keys.ToTufKey(key)
	if len(pk.IDs()) == 0 {
		return nil, errors.New("error getting key ID")
	}
	return &pk.IDs()[0], nil
}

func verifySigningKeys(dirname string, rootCA *x509.Certificate) (*KeyMap, error) {
	// Get all signing keys in the directory.
	log.Printf("\nOutputting key verification and OpenSSL commands...\n")

	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	keyMap := make(KeyMap)
	for _, file := range files {
		if file.IsDir() {
			key, err := keys.SigningKeyFromDir(filepath.Join(dirname, file.Name()))
			if err != nil {
				return nil, err
			}
			if err = key.Verify(rootCA); err != nil {
				log.Printf("error verifying key %d: %s", key.SerialNumber, err)
				return nil, err
			}
			id, err := getKeyID(*key)
			if err != nil {
				return nil, err
			}

			// Note we use relative path here to simplify things.
			wd, err := os.Getwd()
			if err != nil {
				return nil, err
			}
			log.Printf("\nVERIFIED KEY %d\n", key.SerialNumber)
			deviceCert, err := filepath.Rel(wd, filepath.Join(dirname, file.Name(), file.Name()+"_device_cert.pem"))
			if err != nil {
				return nil, err
			}
			keyCert, err := filepath.Rel(wd, filepath.Join(dirname, file.Name(), file.Name()+"_key_cert.pem"))
			if err != nil {
				return nil, err
			}
			log.Printf("\n\t# Manually verify the chain")
			log.Printf("\topenssl verify -verbose -x509_strict -CAfile <(cat piv-attestation-ca.pem %s) %s\n", deviceCert, keyCert)
			log.Printf("\n\t# Manually extract the public key")
			log.Printf("\topenssl x509 -in %s -pubkey -noout", keyCert)
			keyMap[*id] = key
		}
	}
	return &keyMap, nil
}

func verifyMetadata(repository string, keys KeyMap) error {
	log.Printf("\nOutputting metadata verification...\n")

	// logs the state of each metadata file, including number of signatures to achieve threshold
	// and verifies the signatures in each file.
	store := tuf.FileSystemStore(repository, nil)
	db, err := repo.CreateDb(store)
	if err != nil {
		return err
	}
	root, err := repo.GetRootFromStore(store)
	if err != nil {
		return err
	}

	for name, role := range root.Roles {
		log.Printf("\nVerifying %s...", name)
		signed, err := repo.GetSignedMeta(store, name+".json")
		if err != nil {
			return err
		}

		// Rremove the empty placeholder signatures
		var sigs []data.Signature
		for _, sig := range signed.Signatures {
			if len(sig.Signature) != 0 {
				sigs = append(sigs, sig)
			}
		}
		signed.Signatures = sigs

		if err = db.VerifySignatures(signed, name); err != nil {
			if _, ok := err.(verify.ErrRoleThreshold); ok {
				// we may not have all the sig, allow partial sigs for success
				log.Printf("\tContains %d/%d valid signatures\n", err.(verify.ErrRoleThreshold).Actual, role.Threshold)
			} else if err.Error() == verify.ErrNoSignatures.Error() {
				log.Printf("\tContains 0/%d valid signatures\n", role.Threshold)
			} else {
				log.Printf("\tError verifying: %s\n", err)
				return err
			}
		} else {
			log.Printf("\tSuccess! Signatures valid and threshold achieved\n")
		}
	}

	return nil
}

func main() {
	log.SetFlags(0)
	var fileFlag file
	flag.Var(&fileFlag, "root", "Yubico root certificate")
	repository := flag.String("repository", "", "path to repository")
	flag.Parse()

	rootCA, err := toCert(string(fileFlag))
	if err != nil {
		log.Printf("failed to parse root CA: %s", err)
		os.Exit(1)
	}

	if _, err := os.Stat(*repository + "/keys"); os.IsNotExist(err) {
		log.Printf("keys not initialized yet")
		return
	}

	keyMap, err := verifySigningKeys(*repository+"/keys", rootCA)
	if err != nil {
		log.Printf("error verifying signing keys: %s", err)
		os.Exit(1)
	}

	if _, err := os.Stat(*repository + "/staged"); os.IsNotExist(err) {
		if _, err := os.Stat(*repository + "/repository"); os.IsNotExist(err) {
			log.Printf("repository not initialized yet")
			return
		}
	}

	if err := verifyMetadata(*repository, *keyMap); err != nil {
		log.Printf("error verifying signing keys: %s", err)
		os.Exit(1)
	}

}
