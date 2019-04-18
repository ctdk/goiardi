/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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

package organization

import (
	"encoding/gob"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/fakeacl"
	"github.com/ctdk/goiardi/indexer"
	"testing"
)

func init() {
	indexer.Initialize(config.Config)
}

func TestOrgCreation(t *testing.T) {
	z := new(Organization)
	gob.Register(z)
	name := "hlumph"
	fullName := "Hlumphers, Inc."
	o, err := New(name, fullName)
	if err != nil {
		t.Errorf(err.Error())
	}
	if o.Name != name {
		t.Errorf("org names did not match! %s and %s", o.Name, name)
	}
	if o.FullName != fullName {
		t.Errorf("org full name did not match! %s and %s", o.FullName, fullName)
	}
	if len(o.GUID) != 32 {
		t.Errorf("Org GUID should have been 32 characters long, got '%s' of %d chars instead", o.GUID, len(o.GUID))
	}
	err = o.Save()
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestOrgDeletion(t *testing.T) {
	o, err := Get("hlumph")
	if err != nil {
		t.Errorf(err.Error())
	}
	fakeacl.LoadFakeACL(o)
	o.Delete()
	o2, _ := Get("hlumph")
	if o2 != nil {
		t.Errorf("should not have fetched organization, but got '%v' back", o2)
	}
}

func TestOrgGet(t *testing.T) {
	name := "hlumph"
	fullName := "Hlumphers, Inc."
	o, err := New(name, fullName)
	if err != nil {
		t.Errorf(err.Error())
	}
	fakeacl.LoadFakeACL(o)
	err = o.Save()
	o2, err := Get(name)
	if err != nil {
		t.Errorf(err.Error())
	}
	if o.Name != o2.Name {
		t.Errorf("names did not match, got %s and %s", o.Name, o2.Name)
	}
}
