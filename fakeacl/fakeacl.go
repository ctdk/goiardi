/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Package fakeacl implements a fake ACL checker to allow tests to pass and
// avoid dependency loops between orgloader, acl, and actor.
package fakeacl

import (
	"github.com/casbin/casbin/v2"
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/util"
)

type PermTaker interface {
	SetPermCheck(aclhelper.PermChecker)
}

// fake ACL checker for testing
type FakeChecker struct {
}

func (f *FakeChecker) CheckItemPerm(i aclhelper.Item, a aclhelper.Actor, s string) (bool, util.Gerror) {
	return true, nil
}

func (f *FakeChecker) CheckContainerPerm(a aclhelper.Actor, s string, p string) (bool, util.Gerror) {
	return true, nil
}

func (f *FakeChecker) AddMembers(m aclhelper.Role, mm []aclhelper.Member) error {
	return nil
}

func (f *FakeChecker) RemoveMembers(m aclhelper.Role, mm []aclhelper.Member) error {
	return nil
}

func (f *FakeChecker) AddACLRole(m aclhelper.Role) error {
	return nil
}

func (f *FakeChecker) RemoveACLRole(m aclhelper.Role) error {
	return nil
}

func (f *FakeChecker) Enforcer() *casbin.SyncedEnforcer {
	return nil
}

func (f *FakeChecker) RootCheckPerm(a aclhelper.Actor, s string) (bool, util.Gerror) {
	return true, nil
}

func (f *FakeChecker) EditItemPerm(i aclhelper.Item, m aclhelper.Member, perms []string, action string) util.Gerror {
	return nil
}

func (f *FakeChecker) GetItemACL(i aclhelper.Item) (*aclhelper.ACL, error) {
	return nil, nil
}

func (f *FakeChecker) EditFromJSON(i aclhelper.Item, perm string, data interface{}) util.Gerror {
	return nil
}

func (f *FakeChecker) DeleteItemACL(i aclhelper.Item) (bool, error) {
	return false, nil
}

func (f *FakeChecker) RenameItemACL(i aclhelper.Item, s string) error {
	return nil
}

func (f *FakeChecker) CreatorOnly(i aclhelper.Item, a aclhelper.Actor) util.Gerror {
	return nil
}

func (f *FakeChecker) RemoveUser(m aclhelper.Member) error {
	return nil
}

func (f *FakeChecker) RenameMember(m aclhelper.Member, o string) error {
	return nil
}

func (f *FakeChecker) DeletePolicy() error {
	return nil
}

func LoadFakeACL(o PermTaker) {
	f := new(FakeChecker)
	o.SetPermCheck(f)
}
