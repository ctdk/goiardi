/*
 * Copyright (c) 2013-2018, Jeremy Bingham (<jbingham@gmail.com>)
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

package acltesting

import (
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

// group, subkind, kind, name, perm, effect
const (
	condGroupPos = iota
	condSubkindPos
	condKindPos
	condNamePos
	condPermPos
	condEffectPos
)

var pivotal *user.User
var orgCount int

func init() {
	gob.Register(new(organization.Organization))
	gob.Register(new(user.User))
	gob.Register(new(association.Association))
	gob.Register(new(association.AssociationReq))
	gob.Register(new(client.Client))
	gob.Register(new(group.Group))
	gob.Register(new(role.Role))
	gob.Register(make(map[string]interface{}))
	indexer.Initialize(config.Config)
	config.Config.UseAuth = true
}

func setup() {
	confDir, err := ioutil.TempDir("", "acl-test")
	if err != nil {
		panic(err)
	}
	config.Config.PolicyRoot = confDir
	pivotal, _ = user.New("pivotal")
	pivotal.Admin = true
	pivotal.Save()
}

func teardown() {
	os.RemoveAll(config.Config.PolicyRoot)
}

func buildOrg() (*organization.Organization, *user.User) {
	adminUser, _ := user.New(fmt.Sprintf("admin%d", orgCount))
	adminUser.Admin = true
	adminUser.Save()
	org, _ := organization.New(fmt.Sprintf("org%d", orgCount), fmt.Sprintf("test org %d", orgCount))
	orgCount++
	acl.LoadACL(org)
	ar, _ := association.SetReq(adminUser, org, pivotal)
	ar.Accept()
	group.MakeDefaultGroups(org)
	admins, _ := group.Get(org, "admins")
	admins.AddActor(adminUser)
	admins.Save()

	return org, adminUser
}

func TestMain(m *testing.M) {
	setup()
	r := m.Run()
	if r == 0 {
		teardown()
	}
	os.Exit(r)
}

func TestInitACL(t *testing.T) {
	org, _ := organization.New("florp", "mlorph normph")
	acl.LoadACL(org)
	group.MakeDefaultGroups(org)

	/*
		m := casbin.NewModel(modelDefinition)
		e, err := initializeACL(org, m)
		if err != nil {
			t.Error(err)
		}
	*/
	e := org.PermCheck.Enforcer()

	e.AddGroupingPolicy("test1", "role##admins")
	e.AddGroupingPolicy("test_user", "role##users")
	e.SavePolicy()

	testingPolicies := [][]string{
		{"true", "test1", "groups", "containers", "$$default$$", "create", "allow"},
		{"true", "pivotal", "groups", "containers", "$$default$$", "create", "allow"},
		{"true", "test1", "clients", "containers", "$$default$$", "read", "allow"},
		{"false", "test_user", "groups", "containers", "$$default$$", "read", "allow"},
		{"true", "test_user", "roles", "containers", "$$default$$", "read", "allow"},
		{"false", "test_user", "roles", "containers", "$$default$$", "nonexistent_perm", "allow"},
	}

	for _, policy := range testingPolicies {
		var expected bool
		if policy[0] == "true" {
			expected = true
		}
		enforceP := make([]interface{}, len(policy[1:]))
		for i, v := range policy[1:] {
			enforceP[i] = v
		}
		z := e.Enforce(enforceP...)
		if z != expected {
			t.Errorf("Expected '%s' to evaluate as %v, got %v", strings.Join(policy[1:], ", "), expected, z)
		}
	}
	r := e.GetRolesForUser("test1")
	if !util.StringPresentInSlice("role##admins", r) {
		t.Errorf("test1 user should have been a member of the 'admins' group, but wasn't. These roles were found instead: %v", r)
	}
}

func TestCheckItemPerm(t *testing.T) {
	org, adminUser := buildOrg()
	r, _ := role.New(org, "chkitem")
	r.Save()
	chk, err := org.PermCheck.CheckItemPerm(r, adminUser, "create")
	if err != nil {
		t.Errorf("ChkItemPerm for role with adminUser failed: %s", err.Error())
	}
	if !chk {
		t.Errorf("ChkItemPerm for role with adminUser should have been true, but was false.")
	}
	u, _ := user.New("test_user")
	u.Save()
	ar, _ := association.SetReq(u, org, adminUser)
	ar.Accept()
	us, _ := group.Get(org, "users")
	us.AddActor(u)
	us.Save()
	// temporary again
	org.PermCheck.Enforcer().AddGroupingPolicy(u.Username, "role##users")

	chk, err = org.PermCheck.CheckItemPerm(r, u, "create")
	if err != nil {
		t.Errorf("ChkItemPerm for role with normal user failed: %s", err.Error())
	}
	if !chk {
		t.Errorf("ChkItemPerm for role with normal user should have been true, but was false.")
	}
	chk, err = org.PermCheck.CheckItemPerm(r, u, "grant")
	if err != nil {
		t.Errorf("ChkItemPerm for role with normal user failed with an error (should have failed without one): %s", err.Error())
	}
	if chk {
		t.Errorf("ChkItemPerm for role with normal user should have been false, but was true.")
	}

	chk, err = org.PermCheck.CheckItemPerm(r, u, "frobnatz")
	if err == nil {
		t.Error("ChkItemPerm for role with normal user with a non-existent perm failed without an error (should have failed with one)")
	}
	if chk {
		t.Errorf("ChkItemPerm for role with normal user with a non-existent perm should have been false, but was true.")
	}

	chk, err = org.PermCheck.CheckItemPerm(r, adminUser, "frobnatz")
	if err == nil {
		t.Error("ChkItemPerm for role with admin user with a non-existent perm failed without an error (should have failed with one)")
	}
	if chk {
		t.Errorf("ChkItemPerm for role with admin user with a non-existent perm should have been false, but was true.")
	}
}

func TestGroupAdd(t *testing.T) {

}

func TestUserAdd(t *testing.T) {
	org, adminUser := buildOrg()
	u1, _ := user.New("rm_test1")
	u1.Save()
	ar, _ := association.SetReq(u1, org, adminUser)
	ar.Accept()
	us1, _ := group.Get(org, "users")
	u2, _ := user.New("rm_test2")
	u2.Save()
	ar2, _ := association.SetReq(u2, org, adminUser)
	ar2.Accept()
	us2, _ := group.Get(org, "admins")
	us2.AddActor(u2)
	us2.Save()

	// check roles
	r1 := org.PermCheck.Enforcer().GetRolesForUser(u1.Username)
	if !util.StringPresentInSlice(us1.ACLName(), r1) {
		t.Errorf("Role %s not found for %s, got %v instead", us1.ACLName(), u1.GetName(), r1)
	}
	r2 := org.PermCheck.Enforcer().GetRolesForUser(u2.Username)
	if !util.StringPresentInSlice(us2.ACLName(), r2) {
		t.Errorf("Role %s not found for %s, got %v instead", us2.ACLName(), u2.GetName(), r2)
	}
}

func TestGroupRemove(t *testing.T) {

}

func TestUserRemove(t *testing.T) {
	org, adminUser := buildOrg()
	u1, _ := user.New("add_test1")
	u1.Save()
	ar, _ := association.SetReq(u1, org, adminUser)
	ar.Accept()
	u2, _ := user.New("add_test2")
	u2.Save()
	ar2, _ := association.SetReq(u2, org, adminUser)
	ar2.Accept()
	us2, _ := group.Get(org, "admins")
	us2.AddActor(u2)
	us2.Save()

	// make a new group
	gg, _ := group.New(org, "rmgroup")
	gg.Save()
	// add the users in
	gg.AddActor(u1)
	gg.AddActor(u2)
	gg.Save()

	r1 := org.PermCheck.Enforcer().GetRolesForUser(u1.Username)
	if !util.StringPresentInSlice(gg.ACLName(), r1) {
		t.Errorf("Didn't find %s in %s's roles.", gg.ACLName(), u1.Username)
	}
	r2 := org.PermCheck.Enforcer().GetRolesForUser(u2.Username)
	if !util.StringPresentInSlice(gg.ACLName(), r2) {
		t.Errorf("Didn't find %s in %s's roles.", gg.ACLName(), u2.Username)
	}

	err := gg.DelActor(u1)
	if err != nil {
		t.Errorf("error removing %s from %s group: %s", u1.Username, gg.Name, err.Error())
	}
	err = gg.DelActor(u2)
	if err != nil {
		t.Errorf("error removing %s from %s group: %s", u2.Username, gg.Name, err.Error())
	}
	gg.Save()

	r1 = org.PermCheck.Enforcer().GetRolesForUser(u1.Username)
	if util.StringPresentInSlice(gg.ACLName(), r1) {
		t.Errorf("Found %s in %s's roles, when it shouldn't have been.", gg.ACLName(), u1.Username)
	}
	r2 = org.PermCheck.Enforcer().GetRolesForUser(u2.Username)
	if util.StringPresentInSlice(gg.ACLName(), r2) {
		t.Errorf("Found %s in %s's roles, when it shouldn't have been.", gg.ACLName(), u2.Username)
	}
}

func TestClients(t *testing.T) {
	org, _ := buildOrg()
	c, _ := client.New(org, "client1")
	c.Save()

	gg, _ := group.Get(org, "clients")
	gg.AddActor(c)
	gg.Save()

	r1 := org.PermCheck.Enforcer().GetRolesForUser(c.GetName())
	fmt.Printf("roles for client: %v\n", r1)

	if !util.StringPresentInSlice(gg.ACLName(), r1) {
		t.Errorf("Didn't find %s in %s's roles.", gg.ACLName(), c.GetName())
	}
	r, _ := role.New(org, "chkclient")
	r.Save()

	chk, err := org.PermCheck.CheckItemPerm(r, c, "read")
	if err != nil {
		t.Errorf("checking client role read perm failed: %s", err.Error())
	}
	if !chk {
		t.Error("checking a role read perm for a client failed unexpectedly")
	}

	chk, err = org.PermCheck.CheckItemPerm(r, c, "update")
	if err != nil {
		t.Errorf("checking client role update perm failed: %s", err.Error())
	}
	if chk {
		t.Error("checking a role update perm for a client passed unexpectedly")
	}
}

func TestEditItemPerms(t *testing.T) {
	org, adminUser := buildOrg()
	u1, _ := user.New("edit_test1")
	u1.Save()
	ar, _ := association.SetReq(u1, org, adminUser)
	ar.Accept()

	c, _ := client.New(org, "client1")
	c.Save()

	pre, err := org.PermCheck.CheckItemPerm(c, u1, "update")
	if err != nil {
		t.Errorf("checking user client update perm failed: %s", err.Error())
	}
	if pre {
		t.Errorf("client update perm check passed unexpectedly")
	}
	org.PermCheck.EditItemPerm(c, u1, []string{"update"}, "add")
	post, err := org.PermCheck.CheckItemPerm(c, u1, "update")
	if err != nil {
		t.Errorf("checking user client update perm after edit failed: %s", err.Error())
	}
	if !post {
		t.Errorf("client update perm check after edit failed unexpectedly")
	}

	org.PermCheck.EditItemPerm(c, u1, []string{"update"}, "remove")

	removed, err := org.PermCheck.CheckItemPerm(c, u1, "update")
	if err != nil {
		t.Errorf("checking user client update perm after removing failed: %s", err.Error())
	}
	if removed {
		t.Errorf("client update perm check after removing succeeded unexpectedly")
	}

	if merr := org.PermCheck.EditItemPerm(c, u1, []string{"update"}, "flareg"); merr == nil {
		t.Errorf("non-existent action 'flareg' for EditItemPerm did not fail")
	}

	if merr := org.PermCheck.EditItemPerm(c, u1, []string{"glerp"}, "add"); merr == nil {
		t.Errorf("non-existent perm 'glerp' for EditItemPerm did not fail")
	}
}

func TestRootACL(t *testing.T) {
	org, adminUser := buildOrg()
	u1, _ := user.New("root_test1")
	u1.Save()
	ar, _ := association.SetReq(u1, org, adminUser)
	ar.Accept()

	for _, p := range aclhelper.DefaultACLs {
		s, err := org.PermCheck.RootCheckPerm(adminUser, p)
		if !s {
			t.Errorf("Root perm check %s failed for admin user", p)
		}
		if err != nil {
			t.Errorf("Root perm check on %s for admin user had an error: %s", p, err.Error())
		}
	}

	for _, p := range aclhelper.DefaultACLs {
		s, err := org.PermCheck.RootCheckPerm(u1, p)
		if p != "read" {
			if s {
				t.Errorf("Root perm check %s unexpectedly passed for normal user", p)
			}
		} else {
			if !s {
				t.Errorf("Root perm check %s unexpectedly failed for normal user", p)
			}
		}
		if err != nil {
			t.Errorf("Root perm check on %s for normal user had an error: %s", p, err.Error())
		}
	}
}

func TestGetItemAcl(t *testing.T) {
	org, _ := buildOrg()
	r, _ := role.New(org, "getitemacl")
	r.Save()

	m, err := org.PermCheck.GetItemACL(r)
	if err != nil {
		t.Error(err)
	}

	if m == nil {
		t.Error("No elements were found in item ACL")
	} else {
		// Check for the usual suspects in the ACL
		for _, p := range aclhelper.DefaultACLs {
			if a := m.Perms[p].Actors; a == nil {
				t.Errorf("Array for actors in ACL perm '%s' was not found.", p)
			} else {
				if len(a) != 0 {
					t.Errorf("Array of actors in ACL perm '%s' should have had no members, but it had %d members.", p, len(a))
				}
			}
			if g := m.Perms[p].Groups; g == nil {
				t.Errorf("Array for groups in ACL perm '%s' should have been found, but it wasn't!", p)
			} else {
				if len(g) == 0 {
					t.Errorf("Array of groups in ACL perm '%s' should have had members, but it did not.", p)
				} else {
					if g[0] != "admins" {
						t.Errorf("The first member of the Groups array in %s should have been 'role##admins', but was '%s'.", p, g[0])
					}
				}
			}
		}
	}
}

func TestEditFromJSON(t *testing.T) {
	org, adminUser := buildOrg()
	r, _ := role.New(org, "editfromjson")
	r.Save()
	u1, _ := user.New("editjson")
	u1.Save()
	ar, _ := association.SetReq(u1, org, adminUser)
	ar.Accept()

	m, err := org.PermCheck.GetItemACL(r)
	if err != nil {
		t.Error(err)
	}

	mj := m.ToJSON()

	// Man, this is a lot of hoops to jump through to simulate some JSON.
	granters, _ := mj["grant"].(map[string][]string)["actors"]
	granters = append(granters, u1.Name)

	// Something's fishy here.
	groupers, _ := mj["grant"].(map[string][]string)["groups"]

	grantActors := make([]interface{}, len(granters))
	grantGroupers := make([]interface{}, len(groupers))

	for i, v := range granters {
		grantActors[i] = v
	}
	for i, v := range groupers {
		grantGroupers[i] = v
	}

	mj["grant"].(map[string][]string)["actors"] = granters
	grantOp := make(map[string]interface{})
	grantOp["actors"] = grantActors
	grantOp["groups"] = grantGroupers
	jsonContainer := make(map[string]interface{})
	jsonContainer["grant"] = grantOp

	jerr := org.PermCheck.EditFromJSON(r, "grant", jsonContainer)
	if jerr != nil {
		t.Errorf("EditFromJSON failed: %s", jerr.Error())
	}

	permitted, err := org.PermCheck.CheckItemPerm(r, u1, "grant")
	if err != nil {
		t.Errorf("CheckItemPerm after EditFromJSON failed with an error: %s", err.Error())
	}

	if !permitted {
		t.Error("test user should have had 'grant' permission after EditFromJSON, but didn't.")
	}
}

func TestItemDelete(t *testing.T) {
	org, _ := buildOrg()
	c, _ := client.New(org, "itemdelete")
	c.Save()
	us, _ := group.Get(org, "users")
	org.PermCheck.EditItemPerm(c, us, []string{"grant"}, "add")
	cName := c.GetName()
	cKind := c.ContainerKind()
	cType := c.ContainerType()

	pc := org.PermCheck.(*acl.Checker)
	cpol := pc.GetItemPolicies(cName, cKind, cType)
	if cpol == nil || len(cpol) == 0 {
		t.Errorf("policies for client %s should not have been empty", cName)
	} else {
		var found bool
		for _, p := range cpol {
			if p[condPermPos] == "grant" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("policies for client %s should have included 'grant' permission, but didn't.", cName)
		}
	}

	if err := c.Delete(); err != nil {
		t.Errorf("error deleting client during renaming test with ACLs: %s", err.Error())
	}
	delcpol := pc.GetItemPolicies(cName, cKind, cType)
	if delcpol != nil && len(delcpol) != 0 {
		t.Errorf("Deleted client should not have had any policies in the ACL, but it had %d!", len(delcpol))
	}
}

func TestItemRename(t *testing.T) {
	oldName := "itemrename"
	newName := "renamedclient"

	org, _ := buildOrg()
	c, _ := client.New(org, oldName)
	c.Save()
	us, _ := group.Get(org, "users")
	org.PermCheck.EditItemPerm(c, us, []string{"grant"}, "add")

	pc := org.PermCheck.(*acl.Checker)
	cpol := pc.GetItemPolicies(c.Name, c.ContainerKind(), c.ContainerType())
	if cpol == nil || len(cpol) == 0 {
		t.Errorf("policies for client %s should not have been empty", c.Name)
	}

	c.Rename(newName)
	c.Save()

	npol := pc.GetItemPolicies(newName, c.ContainerKind(), c.ContainerType())
	if npol == nil || len(npol) == 0 {
		t.Errorf("policies for renamed client %s (was %s) should not have been empty", c.Name, oldName)
	}
	opol := pc.GetItemPolicies(oldName, c.ContainerKind(), c.ContainerType())
	if opol != nil && len(opol) != 0 {
		t.Errorf("old policies for client %s (was %s) should have been empty, but had %d elements", c.Name, oldName, len(opol))
	}
}

func TestMultipleOrgs(t *testing.T) {

}
