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

// Package chefcrypto bundles up crytographic routines for goairdi.
package chefcrypto

import (
	"strings"
	"testing"
)

/* Incidentally, I have a feeling this test file's going to get a lot more
 * action very soon. */

func TestGenerateRSAKeys(t *testing.T) {
	priv, pub, err := GenerateRSAKeys()
	if err != nil {
		t.Errorf("Generating RSA keys generated an error: %s", err)
	}
	if !strings.HasPrefix(priv, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Errorf("Improper private key: %s", priv)
	}
	if !strings.HasPrefix(pub, "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("Improper public key: %s", pub)
	}
}

var goodPubKey = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApXXeTEd6gq6sziqYt76U\n3W0zT9oBQTdiImrLypUuMTZ6+DBwT4iVLvL/JRydI3tyNRV/S/JZxjPDhaKgnOjZ\ndSVpE9U41DewgB2zzH5mox1LRWTeu8KU3qkuyOTk3qaF3NUejCkkHxKlolLyabRt\npLwaSfQUE+Mr4GoY+gTyo9GSzOKdVoc/PlR+99BDD8AKMm4L705DE1SBHxCRqxfy\n3VLgXfX4GjQKZjH1bDoWSbfaAVOllfB5EDib2IQE58PJwW2milG+XUBbxO0Ee95A\niLx+abdhmVyx6Mysc1Fk9u6VT18pysIi0VwV7SEYu691HfuRT0yEpMZMPb1hFH97\nyQIDAQAB\n-----END PUBLIC KEY-----"

var badPubKey = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApXXeTEd6gq6sziqYt76U\n3W0zT9oBQTdiImrLypUuMTZ6+DBwT4iVLvL/JRydI3tyNRV/S/JZxjPDhaKgnOjZ\ndSVpE9U4ewrWsgB2zzH5mox1LRWTeu8KU3qkuyOTk3qaF3NUejCkkHxKlolLyabRt\npwoahbroE+Mr4GoY+gTyo9GSzOKdVoc/PlR+99BDD8AKMm4L705DE1SBHxCRqxfy\n3VLgXfX4GjQKZjH1bDoWSbfaAVOllfB5EDib2IQE58PJwW2milG+XUBbxO0Ee95A\niLx+abdhmVyx6Mysc1Fk9u6VT18pysIi0VwV7SEYu666HfuRT0yEpMZMPb1hFH97\nyQIDAQAB\n-----END PUBLIC KEY-----"

var oldPubKey = `-----BEGIN RSA PUBLIC KEY-----
MIIBCgKCAQEA61BjmfXGEvWmegnBGSuS+rU9soUg2FnODva32D1AqhwdziwHINFa
D1MVlcrYG6XRKfkcxnaXGfFDWHLEvNBSEVCgJjtHAGZIm5GL/KA86KDp/CwDFMSw
luowcXwDwoyinmeOY9eKyh6aY72xJh7noLBBq1N0bWi1e2i+83txOCg4yV2oVXhB
o8pYEJ8LT3el6Smxol3C1oFMVdwPgc0vTl25XucMcG/ALE/KNY6pqC2AQ6R2ERlV
gPiUWOPatVkt7+Bs3h5Ramxh7XjBOXeulmCpGSynXNcpZ/06+vofGi/2MlpQZNhH
Ao8eayMp6FcvNucIpUndo1X8dKMv3Y26ZQIDAQAB
-----END RSA PUBLIC KEY-----`

var oldLabelPubKey = "-----BEGIN RSA PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApXXeTEd6gq6sziqYt76U\n3W0zT9oBQTdiImrLypUuMTZ6+DBwT4iVLvL/JRydI3tyNRV/S/JZxjPDhaKgnOjZ\ndSVpE9U41DewgB2zzH5mox1LRWTeu8KU3qkuyOTk3qaF3NUejCkkHxKlolLyabRt\npLwaSfQUE+Mr4GoY+gTyo9GSzOKdVoc/PlR+99BDD8AKMm4L705DE1SBHxCRqxfy\n3VLgXfX4GjQKZjH1bDoWSbfaAVOllfB5EDib2IQE58PJwW2milG+XUBbxO0Ee95A\niLx+abdhmVyx6Mysc1Fk9u6VT18pysIi0VwV7SEYu691HfuRT0yEpMZMPb1hFH97\nyQIDAQAB\n-----END PUBLIC KEY-----"

var onePubKey = "1"
var realBadPubKey = "-----BEGIN PUBLIC KEY-----\nI'm bad to the bone\n-----END PUBLIC KEY-----"
var arrPubKey = []string{}
var mapPubKey = make(map[string]interface{})

func TestGoodPubKey(t *testing.T) {
	ok, err := ValidatePublicKey(goodPubKey)
	if !ok {
		t.Errorf("Valid public key was invalid: %s", err.Error())
	}
}

func TestWellFormedBogusPubKey(t *testing.T) {
	ok, _ := ValidatePublicKey(badPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestNumberPubKey(t *testing.T) {
	ok, _ := ValidatePublicKey(onePubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestRealBadPubKey(t *testing.T) {
	ok, _ := ValidatePublicKey(realBadPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestOldPubKey(t *testing.T) {
	ok, err := ValidatePublicKey(oldPubKey)
	if !ok {
		t.Errorf("Old-timey public key did not validate when it should have: %s", err.Error())
	}
}

func TestOldLabeledPubKey(t *testing.T) {
	ok, err := ValidatePublicKey(oldLabelPubKey)
	if !ok {
		t.Errorf("Public key (PKCS#8 but labeled with BEGIN RSA PUBLIC KEY) did not validate when it should have: %s", err.Error())
	}
}

func TestArrayPubKey(t *testing.T) {
	ok, _ := ValidatePublicKey(arrPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestMapPubKey(t *testing.T) {
	ok, _ := ValidatePublicKey(mapPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestHashPasswd(t *testing.T) {
	passwd := "abc123"
	salt := []byte{1, 2, 4, 5, 3, 5, 2, 1, 10}
	nosalt := []byte{}
	expected := "c70b5dd9ebfb6f51d09d4132b7170c9d20750a7852f00680f65658f0310e810056e6763c34c9a00b0e940076f54495c169fc2302cceb312039271c43469507dc"
	saltedExpected := "f4d643377e0809b0a0620bdcb01d7c76b246ee6c19f5d7539ecdbc7d4360b588f0e0254954ece97e9a38a6df6ea72dea4d82166c31ac02415f4e716dfd1b49d0"
	hashedPw, err := HashPasswd(passwd, nosalt)
	if err != nil {
		t.Errorf("Error with unsalted hashed password! %s", err.Error())
	}
	if hashedPw == passwd {
		t.Errorf("password and unsalted hashed password should not be equal")
	}
	if hashedPw != expected {
		t.Errorf("unsalted hashed password was not equal to the expected hash")
	}
	hashedPw, err = HashPasswd(passwd, salt)
	if err != nil {
		t.Errorf("Error with hashed password! %s", err.Error())
	}
	if hashedPw == passwd {
		t.Errorf("password and hashed password should not be equal")
	}
	if hashedPw != saltedExpected {
		t.Errorf("hashed password was not equal to the expected hash")
	}
}
