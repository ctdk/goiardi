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

// Package aclhelper is just an interface definition to allow access to the acl
// methods in various packages that can't import it directly because of import
// cycles.
package aclhelper

import (
	"github.com/casbin/casbin"
	"github.com/ctdk/goiardi/util"
)

var DefaultACLs = [5]string{
	"create",
	"read",
	"update",
	"delete",
	"grant",
}

type Member interface {
	IsACLRole() bool
	ACLName() string
	GetName() string
}

type Role interface {
	IsACLRole() bool
	ACLName() string
	GetName() string
	AllMembers() []Member
}

type Item interface {
	GetName() string
	ContainerKind() string
	ContainerType() string
}

// dummy Item type for root ACLs
type RootACL struct {
	Name    string
	Kind    string
	Subkind string
}

func (r *RootACL) GetName() string {
	return r.Name
}

func (r *RootACL) ContainerKind() string {
	return r.Kind
}

func (r *RootACL) ContainerType() string {
	return r.Subkind
}

// Pretty sure this will be useful in only one or two places, but so it goes.
type ACL struct {
	Name    string
	Kind    string
	Subkind string
	Perms   map[string]*ACLItem
}

type ACLItem struct {
	Perm   string
	Effect string
	Actors []string
	Groups []string
}

// Actor is an interface for objects that can make requests to the server. This
// is a duplicate of the Actor interface in github.com/ctdk/goiardi/actor.
type Actor interface {
	IsAdmin() bool
	IsValidator() bool
	IsSelf(interface{}) bool
	IsUser() bool
	IsClient() bool
	PublicKey() string
	SetPublicKey(interface{}) error
	GetName() string
	CheckPermEdit(map[string]interface{}, string) util.Gerror
	OrgName() string
	ACLName() string
	Authz() string
	IsACLRole() bool
}

type PermChecker interface {
	CheckItemPerm(Item, Actor, string) (bool, util.Gerror)
	CheckContainerPerm(Actor, string, string) (bool, util.Gerror)
	RootCheckPerm(Actor, string) (bool, util.Gerror)
	EditItemPerm(Item, Member, []string, string) util.Gerror
	AddMembers(Role, []Member) error
	RemoveMembers(Role, []Member) error
	AddACLRole(Role) error
	RemoveACLRole(Role) error
	Enforcer() *casbin.SyncedEnforcer
	GetItemACL(Item) (*ACL, error)
	DeleteItemACL(Item) (bool, error)
	RenameItemACL(Item, string) error
	EditFromJSON(Item, string, interface{}) util.Gerror
	CreatorOnly(Item, Actor) util.Gerror
	RemoveUser(Member) error
	RenameMember(Member, string) error
}

// This might be better moved back to the acl module, and made available as
// a module. Hmm.
func (a *ACL) ToJSON() map[string]interface{} {
	jsMap := make(map[string]interface{})
	populate := func(a []string) []string {
		p := make([]string, len(a))
		for i, s := range a {
			p[i] = s
		}
		return p
	}

	for k, v := range a.Perms {
		permData := make(map[string][]string)
		permData["actors"] = populate(v.Actors)
		permData["groups"] = populate(v.Groups)
		jsMap[k] = permData
	}
	return jsMap
}

// this might be useful.
func (a *ACL) Policies() [][]interface{} {
	pols := make([][]interface{}, 0)
	for k, p := range a.Perms {
		s := p.buildPolicyDefs(a.Name, a.Kind, a.Subkind, k)
		pols = append(pols, s...)
	}
	return pols
}

func (i *ACLItem) buildPolicyDefs(name string, kind string, subkind string, perm string) [][]interface{} {
	var policyDefs [][]interface{}
	apol := pslice(name, kind, subkind, perm, i.Effect, i.Actors)
	gpol := pslice(name, kind, subkind, perm, i.Effect, i.Groups)
	policyDefs = append(policyDefs, apol...)
	policyDefs = append(policyDefs, gpol...)

	return policyDefs
}

func pslice(name string, kind string, subkind string, perm string, effect string, subjs []string) [][]interface{} {
	if len(subjs) == 0 {
		return nil
	}
	policies := make([][]interface{}, len(subjs))

	for i, s := range subjs {
		ipol := []interface{}{s, subkind, kind, name, perm, effect}
		policies[i] = ipol
	}
	return policies
}
