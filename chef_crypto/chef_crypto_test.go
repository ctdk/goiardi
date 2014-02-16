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
