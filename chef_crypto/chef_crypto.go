/* Various cryptographic functions, as needed. */

/*
 * Copyright (c) 2013-2014, Jeremy Bingham (<jbingham@gmail.com>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package chef_crypto bundles up crytographic routines for goairdi.
package chef_crypto

import (
	"fmt"
	"crypto/rsa"
	"crypto/rand"
	"encoding/pem"
	"crypto/x509"
)

// Creates a pair of private and public keys for a client.
func GenerateRSAKeys() (string, string, error){
	/* Shamelessly borrowed and adapted from some golang-samples */
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	if err := priv.Validate(); err != nil {
		err_str := fmt.Errorf("RSA key validation failed: %s", err)
		return "", "", err_str
	}
	priv_der := x509.MarshalPKCS1PrivateKey(priv)
	/* For some reason chef doesn't label the keys RSA PRIVATE/PUBLIC KEY */
	priv_blk := pem.Block {
		Type: "RSA PRIVATE KEY",
		Headers: nil,
		Bytes: priv_der,
	}
	priv_pem := string(pem.EncodeToMemory(&priv_blk))
	pub := priv.PublicKey
	pub_der, err := x509.MarshalPKIXPublicKey(&pub)
	if err != nil {
		err_str := fmt.Errorf("Failed to get der format for public key: %s", err)
		return "", "", err_str
	}
	pub_blk := pem.Block {
		Type: "PUBLIC KEY",
		Headers: nil,
		Bytes: pub_der,
	}
	pub_pem := string(pem.EncodeToMemory(&pub_blk))
	return priv_pem, pub_pem, nil
}

func ValidatePublicKey(publicKey interface{}) (bool, error) {
	switch publicKey := publicKey.(type) {
		case string:
			// at the moment we don't care about the pub interface
			decPubKey, _ := pem.Decode([]byte(publicKey))
			if decPubKey == nil {
				err := fmt.Errorf("Public key does not validate")
				return false, err
			}
			if _, err := x509.ParsePKIXPublicKey(decPubKey.Bytes); err != nil {
				nerr := fmt.Errorf("Public key did not validate: %s", err.Error())
				return false, nerr
			}
			return true, nil
		default:
			err := fmt.Errorf("Public key does not validate")
			return false, err
	}
}
