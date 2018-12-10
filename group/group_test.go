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

package group

import (
	"encoding/gob"
	"github.com/casbin/casbin"
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"testing"
)

func init() {
	indexer.Initialize(config.Config)
}

// fake ACL checker for testing
type fakeChecker struct {
}

func (f *fakeChecker) CheckItemPerm(i aclhelper.Item, a aclhelper.Actor, s string) (bool, util.Gerror) {
	return true, nil
}

func (f *fakeChecker) AddMembers(m aclhelper.Role, mm []aclhelper.Member) error {
	return nil
}

func (f *fakeChecker) RemoveMembers(m aclhelper.Role, mm []aclhelper.Member) error {
	return nil
}

func (f *fakeChecker) AddACLRole(m aclhelper.Role) error {
	return nil
}

func (f *fakeChecker) RemoveACLRole(m aclhelper.Role) error {
	return nil
}

func (f *fakeChecker) Enforcer() *casbin.SyncedEnforcer {
	return nil
}

func (f *fakeChecker) RootCheckPerm(a aclhelper.Actor, s string) (bool, util.Gerror) {
	return true, nil
}

func (f *fakeChecker) EditItemPerm(i aclhelper.Item, m aclhelper.Member, perms []string, action string) util.Gerror {
	return nil
}

func (f *fakeChecker) GetItemACL(i aclhelper.Item) (*aclhelper.ACL, error) {
	return nil, nil
}

func (f *fakeChecker) EditFromJSON(i aclhelper.Item, perm string, data interface{}) util.Gerror {
	return nil
}

func (f *fakeChecker) DeleteItemACL(i aclhelper.Item) (bool, error) {
	return false, nil
}

func (f *fakeChecker) RenameItemACL(i aclhelper.Item) error {
	return nil
}

func (f *fakeChecker) CreatorOnly(i aclhelper.Item, a aclhelper.Actor) util.Gerror {
	return nil
}

func (f *fakeChecker) RemoveUser(m aclhelper.Member) error {
	return nil
}

func (f *fakeChecker) RenameMember(m aclhelper.Member, o string) error {
	return nil
}

// More group tests will be coming, as

func TestGroupCreation(t *testing.T) {
	gob.Register(new(organization.Organization))
	gob.Register(new(Group))
	gob.Register(new(user.User))
	org, _ := orgloader.New("florp", "mlorph normph")
	org.PermCheck = &fakeChecker{}

	g, err := New(org, "us0rs")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("group us0rs was unexpectedly nil")
	}
	err = g.Save()
	if err != nil {
		t.Errorf(err.Error())
	}
	g2, err := Get(org, "us0rs")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g2 == nil {
		t.Errorf("refetching group didn't work")
	}
	if g2.Name != g.Name {
		t.Errorf("group names didn't match, expected %s, got %s", g.Name, g2.Name)
	}
}

func TestDefaultGroups(t *testing.T) {
	org, _ := orgloader.New("florp2", "mlorph normph")
	org.PermCheck = &fakeChecker{}
	org.Save()
	u, _ := user.New("pivotal")
	u.Save()
	err := MakeDefaultGroups(org)
	if err != nil {
		t.Errorf(err.Error())
	}

	g, err := Get(org, "users")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group users")
	}
	g, err = Get(org, "admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group admins")
	}
	g, err = Get(org, "billing-admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group billing-admins")
	}
	g, err = Get(org, "clients")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group clients")
	}

}

func TestAddDelActors(t *testing.T) {
	gob.Register(new(user.User))
	org, _ := orgloader.New("florp3", "mlorph normph")
	org.PermCheck = &fakeChecker{}
	org.Save()
	MakeDefaultGroups(org)
	g, _ := Get(org, "users")
	a, _ := user.New("flerkin")
	err := g.AddActor(a)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f, _ := g.checkForActor(a.GetName()); !f {
		t.Errorf("actor %s not found in group after being added", a.GetName())
	}
	err = g.DelActor(a)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f, _ := g.checkForActor(a.GetName()); f {
		t.Errorf("actor %s was found in group after being removed", a.GetName())
	}
}

func TestAddDelGroups(t *testing.T) {
	org, _ := orgloader.New("florp4", "mlorph normph")
	org.PermCheck = &fakeChecker{}
	org.Save()
	MakeDefaultGroups(org)
	g, _ := Get(org, "admins")
	a, _ := New(org, "mlerkle")
	err := g.AddGroup(a)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f, _ := g.checkForGroup(a.Name); !f {
		t.Errorf("group %s not found in group after being added", a.Name)
	}
	err = g.DelGroup(a)
	if err != nil {
		t.Errorf(err.Error())
	}
	if f, _ := g.checkForActor(a.Name); f {
		t.Errorf("group %s was found in group after being removed", a.Name)
	}
}

func TestSeekActor(t *testing.T) {
	org, _ := orgloader.New("florp5", "mlorph normph")
	org.PermCheck = &fakeChecker{}
	org.Save()
	MakeDefaultGroups(org)
	g, _ := Get(org, "admins")
	a, _ := user.New("gurbur")
	err := g.AddActor(a)
	if err != nil {
		t.Error(err)
	}
	tt := g.SeekActor(a)
	if !tt {
		t.Errorf("SeekActor failed to find %s in the %s group", a.Name, g.Name)
	}
}
