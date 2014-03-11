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
	"testing"
	"strings"
)

/* Incidentally, I have a feeling this test file's going to get a lot more 
 * action very soon. */

func TestGenerateRSAKeys(t *testing.T){
	priv, pub, err := GenerateRSAKeys()
	if err != nil {
		t.Errorf("Generating RSA keys generated an error: %s", err)
	}
	if !strings.HasPrefix(priv, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Errorf("Improper private key: %s", priv)
	}
	if !strings.HasPrefix(pub, "-----BEGIN PUBLIC KEY-----"){
		t.Errorf("Improper public key: %s", pub)
	}
}

var goodPubKey = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApXXeTEd6gq6sziqYt76U\n3W0zT9oBQTdiImrLypUuMTZ6+DBwT4iVLvL/JRydI3tyNRV/S/JZxjPDhaKgnOjZ\ndSVpE9U41DewgB2zzH5mox1LRWTeu8KU3qkuyOTk3qaF3NUejCkkHxKlolLyabRt\npLwaSfQUE+Mr4GoY+gTyo9GSzOKdVoc/PlR+99BDD8AKMm4L705DE1SBHxCRqxfy\n3VLgXfX4GjQKZjH1bDoWSbfaAVOllfB5EDib2IQE58PJwW2milG+XUBbxO0Ee95A\niLx+abdhmVyx6Mysc1Fk9u6VT18pysIi0VwV7SEYu691HfuRT0yEpMZMPb1hFH97\nyQIDAQAB\n-----END PUBLIC KEY-----"

var badPubKey = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApXXeTEd6gq6sziqYt76U\n3W0zT9oBQTdiImrLypUuMTZ6+DBwT4iVLvL/JRydI3tyNRV/S/JZxjPDhaKgnOjZ\ndSVpE9U4ewrWsgB2zzH5mox1LRWTeu8KU3qkuyOTk3qaF3NUejCkkHxKlolLyabRt\npwoahbroE+Mr4GoY+gTyo9GSzOKdVoc/PlR+99BDD8AKMm4L705DE1SBHxCRqxfy\n3VLgXfX4GjQKZjH1bDoWSbfaAVOllfB5EDib2IQE58PJwW2milG+XUBbxO0Ee95A\niLx+abdhmVyx6Mysc1Fk9u6VT18pysIi0VwV7SEYu666HfuRT0yEpMZMPb1hFH97\nyQIDAQAB\n-----END PUBLIC KEY-----"

var onePubKey = "1"
var realBadPubKey = "-----BEGIN PUBLIC KEY-----\nI'm bad to the bone\n-----END PUBLIC KEY-----"
var arrPubKey = []string{}
var mapPubKey = make(map[string]interface{})

func TestGoodPubKey(t *testing.T){
	ok, err := ValidatePublicKey(goodPubKey)
	if !ok {
		t.Errorf("Valid public key was invalid: %s", err.Error())
	}
}

func TestWellFormedBogusPubKey(t *testing.T){
	ok, _ := ValidatePublicKey(badPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestNumberPubKey(t *testing.T){
	ok, _ := ValidatePublicKey(onePubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestRealBadPubKey(t *testing.T){
	ok, _ := ValidatePublicKey(realBadPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestArrayPubKey(t *testing.T){
	ok, _ := ValidatePublicKey(arrPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestMapPubKey(t *testing.T){
	ok, _ := ValidatePublicKey(mapPubKey)
	if ok {
		t.Errorf("Well-formed but bogus public key validated when it should not have.")
	}
}

func TestHashPasswd(t *testing.T){
	passwd := "abc123"
	expected := "c70b5dd9ebfb6f51d09d4132b7170c9d20750a7852f00680f65658f0310e810056e6763c34c9a00b0e940076f54495c169fc2302cceb312039271c43469507dc"
	hashedPw, err := HashPasswd(passwd)
	if err != nil {
		t.Errorf("Error with hashed password! %s", err.Error())
	}
	if hashedPw == passwd {
		t.Errorf("password and hashed password should not be equal")
	}
	if hashedPw != expected {
		t.Errorf("hashed password was not equal to the expected hash")
	}
}
