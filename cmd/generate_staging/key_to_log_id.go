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

package main

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
)

var (
	key = flag.String("key", "", "Base-64 encoded DER encoded key (PEM without headers and newlines), either PKCS1 or PKIX formatted")
)

func main() {
	flag.Parse()

	if *key == "" {
		log.Fatal("--key must be set")
	}

	var err error
	decodedKey, err := base64.StdEncoding.DecodeString(*key)
	if err != nil {
		log.Fatal("key not base64 encoded")
	}

	var pubkey crypto.PublicKey
	pubkey, err = x509.ParsePKCS1PublicKey(decodedKey)
	if err != nil {
		pubkey, err = x509.ParsePKIXPublicKey(decodedKey)
		if err != nil {
			log.Fatal("invalid public key")
		}
	}

	logID, err := getLogID(pubkey)
	if err != nil {
		log.Fatal("could not generate log ID")
	}
	fmt.Println(logID)
}

func getLogID(pub crypto.PublicKey) (string, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(pubBytes)
	return base64.StdEncoding.EncodeToString(digest[:]), nil
}
