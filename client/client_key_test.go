/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Client key tests
package client

import (
	"encoding/gob"
	"github.com/ctdk/chefcrypto"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/fakeacl"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/infinity"
	"github.com/ctdk/goiardi/organization"
	"testing"
	"time"
)

var keyOrg *organization.Organization

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(organization.Organization))
	gob.Register(new(Client))
	gob.Register(new(Key))
	gob.Register(make(map[string]*Key))
	keyOrg, _ = organization.New("defaultkey", "boo")
	fakeacl.LoadFakeACL(keyOrg)
	keyOrg.Save()
	indexer.Initialize(config.Config, keyOrg)
}

func TestGenerateNamedKeys(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_1")
	c.Save()

	priv, key, err := c.GenerateNamedKeys("default", infinity.Infinity)
	if err != nil {
		t.Errorf("generating default key failed: %s", err.Error())
	}
	if key.Name != "default" {
		t.Errorf("key.Name should have been 'default', was '%s'", key.Name)
	}
	if key.ExpirationDate != infinity.Infinity {
		t.Errorf("key.ExpirationDate should have (essentially) been infinite, but was %v instead", key.ExpirationDate)
	}
	if priv == "" {
		t.Error("private key should not have been empty, but was")
	}

}

func TestKeyZeroExpirationDate(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_2")
	c.Save()
	var tt time.Time
	priv, key, err := c.GenerateNamedKeys("default", tt)
	if err != nil {
		t.Errorf("generating default key with zero expiration date failed: %s", err.Error())
	}
	if key.Name != "default" {
		t.Errorf("key.Name should have been 'default', was '%s'", key.Name)
	}
	if key.ExpirationDate != infinity.Infinity {
		t.Errorf("key.ExpirationDate should have (essentially) been infinite, but was %v instead", key.ExpirationDate)
	}
	if priv == "" {
		t.Error("private key should not have been empty, but was")
	}

}

func TestKeySpecificExpirationDate(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_3")
	c.Save()
	tt := time.Now().Add(time.Hour * 24 * 30)
	priv, key, err := c.GenerateNamedKeys("default", tt)
	if err != nil {
		t.Errorf("generating default key with specific expiration date failed: %s", err.Error())
	}
	if key.Name != "default" {
		t.Errorf("key.Name should have been 'default', was '%s'", key.Name)
	}
	if key.ExpirationDate != tt {
		t.Errorf("key.ExpirationDate should have (essentially) been infinite, but was %v instead", key.ExpirationDate)
	}
	if priv == "" {
		t.Error("private key should not have been empty, but was")
	}

}


func TestGetKey(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_4")
	c.Save()

	_, _, err := c.GenerateNamedKeys("default", infinity.Infinity)
	if err != nil {
		t.Errorf("generating default key failed: %s", err.Error())
	}

	keys := c.GetAllPublicKeys()
	if len(keys) != 1 {
		t.Errorf("GetAllPublicKeys should have returned 1 item, but instead returned %d", len(keys))
	}
	if k, ok := keys["default"]; !ok {
		t.Error("keys[\"default\"] should have existed, but was empty")
	} else {
		if k.Name != "default" {
			t.Errorf("k.Name should have been 'default', but was %s", k.Name)
		}
		if k.ExpirationDate != infinity.Infinity {
			t.Errorf("k.ExpirationDate should have been essentially infinite, but was %v instead", k.ExpirationDate)
		}
	}
}

func TestUpdatingKey(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_5")
	c.Save()

	_, _, err := c.GenerateNamedKeys("default", infinity.Infinity)
	if err != nil {
		t.Errorf("generating default key failed: %s", err.Error())
	}

	keys := c.GetAllPublicKeys()
	if len(keys) != 1 {
		t.Errorf("UpdatingKey: GetAllPublicKeys should have returned 1 item, but instead returned %d", len(keys))
	}
	if k, ok := keys["default"]; !ok {
		t.Error("UpdatingKey: keys[\"default\"] should have existed, but was empty")
	} else {
		if k.Name != "default" {
			t.Errorf("UpdatingKey: k.Name should have been 'default', but was %s", k.Name)
		}
		if k.ExpirationDate != infinity.Infinity {
			t.Errorf("UpdatingKey: k.ExpirationDate should have been essentially infinite at the beginning, but was %v instead", k.ExpirationDate)
		}

		// update key
		xt := time.Now().Add(time.Hour * 24)
		k.ExpirationDate = xt
		if xerr := c.SetNamedKey(k); xerr != nil {
			t.Errorf("Updating key error: %s", xerr.Error())
		}
		nk := c.NamedPublicKey("default")
		if nk.ExpirationDate == infinity.Infinity {
			t.Errorf("default key's expiration date should have been updated to %s, but was still infinite", xt)
		}
		if !nk.ExpirationDate.Equal(xt) {
			t.Errorf("default key's expiration date should have been %s, but was %s instead", xt, nk.ExpirationDate)
		}
	}
}

func TestMultipleKey(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_6")
	c.Save()

	keyNames := []string{"default", "foober", "noomer"}

	for _, kName := range keyNames {
		_, _, err := c.GenerateNamedKeys(kName, infinity.Infinity)
		if err != nil {
			t.Errorf("generating %s key failed: %s", kName, err.Error())
		}
	}

	keys := c.GetAllPublicKeys()
	if len(keys) != len(keyNames) {
		t.Errorf("GetAllPublicKeys should have returned %d item, but instead returned %d", len(keyNames), len(keys))
	}

	for _, kName := range keyNames {
		if k, ok := keys[kName]; !ok {
			t.Errorf("keys[\"%s\"] should have existed, but was empty", kName)
		} else {
			if k.Name != kName {
				t.Errorf("k.Name should have been '%s', but was %s", kName, k.Name)
			}
			if k.ExpirationDate != infinity.Infinity {
				t.Errorf("%s k.ExpirationDate should have been essentially infinite, but was %v instead", kName, k.ExpirationDate)
			}
		}
	}
}

func TestKeysArray(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_7")
	c.Save()

	keyNames := []string{"default", "foober", "noomer"}

	for _, kName := range keyNames {
		_, _, err := c.GenerateNamedKeys(kName, infinity.Infinity)
		if err != nil {
			t.Errorf("generating %s key failed: %s", kName, err.Error())
		}
	}
	keys := c.GetKeysArray()
	if len(keys) != len(keyNames) {
		t.Errorf("array of keys returned should have had len(%d), but had len(%d) instead", len(keyNames), len(keys))
	}
}

func TestKeyDelete(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_8")
	c.Save()

	keyNames := []string{"default", "foober", "noomer"}
	for _, kName := range keyNames {
		_, _, err := c.GenerateNamedKeys(kName, infinity.Infinity)
		if err != nil {
			t.Errorf("generating %s key failed: %s", kName, err.Error())
		}
	}
	c.DeleteKey("noomer")

	keys := c.GetAllPublicKeys()
	if len(keys) != len(keyNames)-1 {
		t.Errorf("len(keys) should have been %d after deleting one key, but was %d instead", len(keyNames)-1, len(keys))
	}

	k := c.NamedPublicKey("noomer")
	if k != nil {
		t.Errorf("Attempting to fetch 'noomer' should have returned a nil key, but instead returned %v", k)
	}
}

func TestKeyDeleteAll(t *testing.T) {
	c, _ := New(keyOrg, "foo_key_9")
	c.Save()

	keyNames := []string{"default", "foober", "noomer"}
	for _, kName := range keyNames {
		_, _, err := c.GenerateNamedKeys(kName, infinity.Infinity)
		if err != nil {
			t.Errorf("generating %s key failed: %s", kName, err.Error())
		}
	}
	c.DeleteAllKeys()

	keys := c.GetAllPublicKeys()
	if len(keys) != 0 {
		t.Errorf("len(keys) should have been 0 after deleting all, but was %d instead", len(keys))
	}

	k := c.NamedPublicKey("noomer")
	if k != nil {
		t.Errorf("Attempting to fetch 'noomer' should have returned a nil key, but instead returned %v", k)
	}
}

// simulate creating a key from a JSON blob
func TestKeyFromJSON(t *testing.T) {
	j := make(map[string]interface{})
	j["expiration_date"] = "infinity"
	j["name"] = "from_json"
	_, pubPem, _ := chefcrypto.GenerateRSAKeys()
	j["public_key"] = pubPem

	k, err := KeyFromJSON(j)
	if err != nil {
		t.Errorf("error creating public key from JSON stand-in: %s", err.Error())
	}
	if k.ExpirationDate != infinity.Infinity {
		t.Errorf("KeyFromJSON expiration should have been (essentially) infinite, but was %v", k.ExpirationDate)
	}
	if k.Name != "from_json" {
		t.Errorf("KeyFromJSON name should have been 'from_json', but was '%s'", k.Name)
	}
}
