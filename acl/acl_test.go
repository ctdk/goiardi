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

package acl

import (
	"encoding/gob"
	"testing"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
)

func TestDefaultACLs(t *testing.T) {
	gob.Register(new(organization.Organization))
	gob.Register(new(group.Group))
	gob.Register(new(ACL))
	gob.Register(new(ACLitem))
	org, _ := organization.New("florp", "mlorph normph")
	group.MakeDefaultGroups(org)
	a, err := Get(org, "groups", "admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	if a.ACLitems["create"].Groups[0].Name != "admins" {
		t.Errorf("group in create group wrong, expected 'admins', got '%s'",  a.ACLitems["create"].Groups[0].Name)
	}
}

func TestAddGroupToACL(t *testing.T) {
	org, _ := organization.New("florp2", "mlorph normph")
	group.MakeDefaultGroups(org)
	g, _ := group.New(org, "fooper")
	a, err := Get(org, "groups", "admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	err = a.AddGroup("create", g)
	if err != nil {
		t.Errorf(err.Error())
	}
	err = a.Save()
	var f bool
	for _, y := range a.ACLitems["create"].Groups {
		if y.Name == g.Name {
			f = true
		}
	}
	if !f {
		t.Errorf("adding group %s to acl failed", g.Name)
	}
	if err != nil {
		t.Errorf(err.Error())
	}
	a2, err := Get(org, "groups", "admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	if a2.Kind != a.Kind || a2.Subkind != a.Subkind {
		t.Errorf("ACLs did not match, expected '%s/%s', got '%s/s'", a.Kind, a.Subkind, a2.Kind, a2.Subkind)
	}
}
