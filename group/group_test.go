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

package group

import (
	"encoding/gob"
	"fmt"
	"testing"

	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/fakeacl"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
)

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
}

var groupOrgCounter int

func setupTestGroupOrg(t *testing.T) *organization.Organization {
	groupOrgCounter++
	org, err := organization.New(fmt.Sprintf("group-org-%d", groupOrgCounter), "Test Group Org")
	if err != nil {
		t.Fatalf("failed to create org: %s", err.Error())
	}
	fakeacl.LoadFakeACL(org)
	if err := org.Save(); err != nil {
		t.Fatalf("failed to save org: %s", err.Error())
	}
	return org
}
func setupTestGroupOrgWithDefaults(t *testing.T) *organization.Organization {
	org := setupTestGroupOrg(t)
	if _, err := user.Get("pivotal"); err != nil {
		u, err := user.New("pivotal")
		if err != nil {
			t.Fatalf("failed to create pivotal user: %s", err.Error())
		}
		if err := u.Save(); err != nil {
			t.Fatalf("failed to save pivotal user: %s", err.Error())
		}
	}
	if err := MakeDefaultGroups(org); err != nil {
		t.Fatalf("failed to make default groups: %s", err.Error())
	}
	return org
}


// More group tests will be coming, as

func TestGroupCreation(t *testing.T) {
	gob.Register(new(organization.Organization))
	gob.Register(new(Group))
	gob.Register(new(user.User))
	org := setupTestGroupOrg(t)

	g, err := New(org, "us0rs")
	if err != nil {
		t.Error(err.Error())
	}
	if g == nil {
		t.Error("group us0rs was unexpectedly nil")
	}
	err = g.Save()
	if err != nil {
		t.Error(err.Error())
	}
	g2, err := Get(org, "us0rs")
	if err != nil {
		t.Error(err.Error())
	}
	if g2 == nil {
		t.Error("refetching group didn't work")
	}
	if g2.Name != g.Name {
		t.Errorf("group names didn't match, expected %s, got %s", g.Name, g2.Name)
	}
}

func TestDefaultGroups(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)

	g, err := Get(org, "users")
	if err != nil {
		t.Error(err.Error())
	}
	if g == nil {
		t.Error("failed to get created default group users")
	}
	if f, _ := g.checkForActor(DefaultUser); !f {
		t.Errorf("failed to find pivotal user in %s", g.Name)
	}

	g, err = Get(org, "admins")
	if err != nil {
		t.Error(err.Error())
	}
	if g == nil {
		t.Error("failed to get created default group admins")
	}
	if f, _ := g.checkForActor(DefaultUser); !f {
		t.Errorf("failed to find pivotal user in %s", g.Name)
	}

	g, err = Get(org, "billing-admins")
	if err != nil {
		t.Error(err.Error())
	}
	if g == nil {
		t.Error("failed to get created default group billing-admins")
	}
	g, err = Get(org, "clients")
	if err != nil {
		t.Error(err.Error())
	}
	if g == nil {
		t.Error("failed to get created default group clients")
	}

}

func TestAddDelActors(t *testing.T) {
	gob.Register(new(user.User))
	org := setupTestGroupOrgWithDefaults(t)
	g, _ := Get(org, "users")
	a, _ := user.New("flerkin")
	err := g.AddActor(a)
	if err != nil {
		t.Error(err.Error())
	}
	if f, _ := g.checkForActor(a.GetName()); !f {
		t.Errorf("actor %s not found in group after being added", a.GetName())
	}
	err = g.DelActor(a)
	if err != nil {
		t.Error(err.Error())
	}
	if f, _ := g.checkForActor(a.GetName()); f {
		t.Errorf("actor %s was found in group after being removed", a.GetName())
	}
}

func TestAddDelGroups(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	g, _ := Get(org, "admins")
	a, _ := New(org, "mlerkle")
	err := g.AddGroup(a)
	if err != nil {
		t.Error(err.Error())
	}
	if f, _ := g.checkForGroup(a.Name); !f {
		t.Errorf("group %s not found in group after being added", a.Name)
	}
	err = g.DelGroup(a)
	if err != nil {
		t.Error(err.Error())
	}
	if f, _ := g.checkForActor(a.Name); f {
		t.Errorf("group %s was found in group after being removed", a.Name)
	}
}

func TestSeekActor(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	g, _ := Get(org, "admins")
	a, _ := user.New("gurbur")
	err := g.AddActor(a)
	if err != nil {
		t.Error(err)
	}
	tt := g.SeekActor(a)
	if !tt {
		t.Errorf("SeekActor failed to find %s in the %s group", a.Username, g.Name)
	}
}

func TestGroupAllGroups(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	g, err := New(org, "extra-group")
	if err != nil {
		t.Fatalf("failed to create extra group: %s", err.Error())
	}
	if err := g.Save(); err != nil {
		t.Fatalf("failed to save extra group: %s", err.Error())
	}
	groups := AllGroups(org)
	if len(groups) < 5 {
		t.Errorf("expected at least 5 groups, got %d", len(groups))
	}
}

func TestGroupGetList(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	list := GetList(org)
	if len(list) < 4 {
		t.Errorf("expected at least 4 groups, got %d", len(list))
	}
}

func TestGroupClearActor(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	a, _ := user.New("clear-me")
	a.Save()
	admins, _ := Get(org, "admins")
	users, _ := Get(org, "users")
	admins.AddActor(a)
	users.AddActor(a)
	admins.Save()
	users.Save()

	ClearActor(org, actor.Actor(a))

	admins, _ = Get(org, "admins")
	users, _ = Get(org, "users")
	if admins.SeekActor(actor.Actor(a)) {
		t.Error("expected actor to be cleared from admins")
	}
	if users.SeekActor(actor.Actor(a)) {
		t.Error("expected actor to be cleared from users")
	}
}

func TestGroupToJSON(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	g, _ := Get(org, "admins")
	j := g.ToJSON()
	if j["name"] != "admins" {
		t.Errorf("expected admins, got %v", j["name"])
	}
}

func TestGroupGroupsByIdSQLError(t *testing.T) {
	org := setupTestGroupOrg(t)
	_, err := GroupsByIdSQL([]int64{1}, org)
	if err == nil {
		t.Fatal("expected error from GroupsByIdSQL without DB backend")
	}
	if err.Error() != "GroupsByIdSQL only works if you're using a database storage backend." {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestGroupRename(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	g, err := New(org, "oldname")
	if err != nil {
		t.Fatalf("failed to create group: %s", err.Error())
	}
	if err := g.Save(); err != nil {
		t.Fatalf("failed to save group: %s", err.Error())
	}
	if err := g.Rename("newname"); err != nil {
		t.Fatalf("Rename() failed: %s", err.Error())
	}
	if _, err := Get(org, "oldname"); err == nil {
		t.Error("expected old group name to be unavailable after rename")
	}
	g2, err := Get(org, "newname")
	if err != nil {
		t.Fatalf("failed to get renamed group: %s", err.Error())
	}
	if g2.Name != "newname" {
		t.Errorf("expected newname, got %s", g2.Name)
	}
}

func TestGroupDelete(t *testing.T) {
	org := setupTestGroupOrgWithDefaults(t)
	g, err := New(org, "deleteme")
	if err != nil {
		t.Fatalf("failed to create group: %s", err.Error())
	}
	if err := g.Save(); err != nil {
		t.Fatalf("failed to save group: %s", err.Error())
	}
	if err := g.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	if _, err := Get(org, "deleteme"); err == nil {
		t.Error("expected deleted group to be gone")
	}
}

func TestGroupAllMembers(t *testing.T) {
	gob.Register(new(user.User))
	org := setupTestGroupOrgWithDefaults(t)
	g, _ := New(org, "member-test")
	u, _ := user.New("member-user")
	nested, _ := New(org, "nested-group")
	g.AddActor(u)
	g.AddGroup(nested)
	members := g.AllMembers()
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestGroupContainerTypeAndACLName(t *testing.T) {
	org := setupTestGroupOrg(t)
	g, _ := New(org, "aclname-test")
	if g.ContainerType() != "$$default$$" {
		t.Errorf("expected $$default$$, got %s", g.ContainerType())
	}
	if g.ACLName() != "role##aclname-test" {
		t.Errorf("unexpected ACL name: %s", g.ACLName())
	}
	if !g.IsACLRole() {
		t.Error("expected group to be an ACL role")
	}
}
