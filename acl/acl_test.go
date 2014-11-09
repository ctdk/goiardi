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
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"testing"
)

func TestDefaultACLs(t *testing.T) {
	gob.Register(new(organization.Organization))
	gob.Register(new(group.Group))
	gob.Register(new(ACL))
	gob.Register(new(ACLitem))
	gob.Register(new(user.User))
	u, _ := user.New("pivotal")
	u.Admin = true
	u.Save()
	org, _ := organization.New("florp", "mlorph normph")
	group.MakeDefaultGroups(org)
	a, err := Get(org, "groups", "admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	if a.ACLitems["create"].Groups[0].Name != "admins" {
		t.Errorf("group in create group wrong, expected 'admins', got '%s'", a.ACLitems["create"].Groups[0].Name)
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

func TestUserPermCheck(t *testing.T) {
	org, _ := organization.New("florp3", "mlorph normph")
	group.MakeDefaultGroups(org)
	a, _ := Get(org, "groups", "admins")
	u, _ := user.New("moohoo")
	u.Save()
	err := a.AddActor("create", u)
	if err != nil {
		t.Errorf(err.Error())
	}
	f, err := a.CheckPerm("create", u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if !f {
		t.Errorf("Perm check didn't work!")
	}
	f, err = a.CheckPerm("update", u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f {
		t.Errorf("Perm check succeeded when it should not have")
	}
}

func TestClientPermCheck(t *testing.T) {
	org, _ := organization.New("florp4", "mlorph normph")
	group.MakeDefaultGroups(org)
	a, _ := Get(org, "groups", "admins")
	gob.Register(new(client.Client))
	c, _ := client.New(org, "moom")
	c.Save()
	err := a.AddActor("create", c)
	if err != nil {
		t.Errorf(err.Error())
	}
	f, err := a.CheckPerm("create", c)
	if err != nil {
		t.Errorf(err.Error())
	}
	if !f {
		t.Errorf("Client perm check didn't work!")
	}
	f, err = a.CheckPerm("update", c)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f {
		t.Errorf("Client perm check succeeded when it should not have")
	}
}

func TestGroupPermCheck(t *testing.T) {
	org, _ := organization.New("florp5", "mlorph normph")
	group.MakeDefaultGroups(org)
	u, _ := user.New("moohoo2")
	a, _ := Get(org, "groups", "admins")
	g, _ := group.New(org, "mnerg")
	g.AddActor(u)
	err := a.AddGroup("create", g)
	if err != nil {
		t.Errorf(err.Error())
	}
	f, err := a.CheckPerm("create", u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if !f {
		t.Errorf("Group perm check didn't work!")
	}
	f, err = a.CheckPerm("update", u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f {
		t.Errorf("Group perm check succeeded when it should not have")
	}
}

func TestMultiLevelGroupPermCheck(t *testing.T) {
	org, _ := organization.New("florp6", "mlorph normph")
	group.MakeDefaultGroups(org)
	u, _ := user.New("moohoo3")
	a, _ := Get(org, "groups", "admins")
	g, _ := group.New(org, "mnergor")
	g.AddActor(u)
	g2, _ := group.New(org, "flermern")
	g2.AddGroup(g)
	err := a.AddGroup("create", g2)
	if err != nil {
		t.Errorf(err.Error())
	}
	f, err := a.CheckPerm("create", u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if !f {
		t.Errorf("Group perm check didn't work!")
	}
	f, err = a.CheckPerm("update", u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f {
		t.Errorf("Group perm check succeeded when it should not have")
	}
}
