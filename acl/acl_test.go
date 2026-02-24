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

// Tests for ACLs. Ugh.
package acl

import (
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"testing"
)

type fakeItem struct {
	name string
	kind string
	ctype string
}

func (f *fakeItem) GetName() string {
	return f.name
}

func (f *fakeItem) ContainerKind() string {
	return f.kind
}

func (f *fakeItem) ContainerType() string {
	return f.ctype
}

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
}

func TestOrgACL(t *testing.T) {
	gob.Register(new(organization.Organization))
	org, _ := organization.New("leep", "sdfasdf")
	org.Save()

	aclErr := LoadACL(org)
	if aclErr != nil {
		t.Error(aclErr)
	}
	c := &fakeItem{"clients", "containers", "containers"}
	a, err := org.PermCheck.GetItemACL(c)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("acl: %v\n", a)
}
