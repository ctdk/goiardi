/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Some client tests, for now.
package client

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"testing"
)

func TestGobEncodeDecode(t *testing.T) {
	indexer.Initialize(config.Config)
	c, _ := New("foo")
	saved := new(bytes.Buffer)
	var err error
	enc := gob.NewEncoder(saved)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Something went wrong encoding the data store with Gob")
		}
	}()
	err = enc.Encode(c)
	if err != nil {
		t.Error(err.Error())
	}
	dec := gob.NewDecoder(saved)
	c2 := new(Client)
	err = dec.Decode(&c2)
	if err != nil {
		t.Error(err.Error())
	}
	if c2.Name != c.Name {
		t.Errorf("saved user doesn't seem to be equal to original: %v vs %v", c2, c)
	}
}

func TestActionAtADistance(t *testing.T) {
	c, _ := New("foo2")
	gob.Register(c)
	c.Save()
	c2, _ := Get("foo2")
	if c.Name != c2.Name {
		t.Errorf("Client names should have been the same, but weren't, got %s and %s", c.Name, c2.Name)
	}
	c2.Validator = true
	if c.Validator == c2.Validator {
		t.Errorf("Changing the value of validator on one client improperly changed it on the other")
	}
}

// TestUpdateFromJSONPreservesFlags guards against the admin/validator flags
// being silently reset when an update omits those keys.
func TestUpdateFromJSONPreservesFlags(t *testing.T) {
	c, _ := New("flagtest")
	c.Admin = true

	// An update that omits "admin" and "validator" must leave the existing
	// flag values untouched.
	upd := map[string]interface{}{
		"name":       "flagtest",
		"json_class": "Chef::ApiClient",
		"chef_type":  "client",
	}
	if err := c.UpdateFromJSON(upd); err != nil {
		t.Fatalf("UpdateFromJSON returned an error: %s", err.Error())
	}
	if !c.Admin {
		t.Errorf("admin flag was reset to false when the 'admin' key was omitted from the update")
	}
	if c.Validator {
		t.Errorf("validator flag unexpectedly became true")
	}

	// Supplying a flag explicitly must still take effect.
	c2, _ := New("flagtest2")
	upd2 := map[string]interface{}{
		"name":       "flagtest2",
		"json_class": "Chef::ApiClient",
		"chef_type":  "client",
		"validator":  true,
	}
	if err := c2.UpdateFromJSON(upd2); err != nil {
		t.Fatalf("UpdateFromJSON returned an error: %s", err.Error())
	}
	if !c2.Validator {
		t.Errorf("validator flag was not set to true when supplied")
	}
}
