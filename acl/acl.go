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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
)

var DefaultACLs = [5]string{
	"create",
	"read",
	"update",
	"delete",
	"grant",
}

type ACLOwner interface {
}

type ACLitem struct {
	Perm   string
	Actors []actor.Actor
	Groups []*group.Group
}

type ACL struct {
	ACLitems map[string]*ACLitem
	Owner    ACLOwner
	isModified bool
}

func defaultACL(org *organization.Organization, kind string, subkind string) *ACL {
	acl := make(ACL)
	// almost always we'd want these default acls
	acl.ACLitems = make(map[string]*ACLitem)
	for _, a := range DefaultACLs {
		acl.ACLitems[a] = &ACLitem{Perm: a}
	}
	switch kind {
	case "containers":
		switch subkind {
		case "$$root$$", "containers", "groups":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["read"], "users")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["grant"], "admins")	
		case "cookbooks", "environments", "roles":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["create"], "users")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["read"], "users")
			addGroup(org, acl.ACLitems["read"], "clients")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["update"], "users")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["delete"], "users")
			addGroup(org, acl.ACLitems["grant"], "admins")
		// bit confusing here: chef-zero says cookbooks have both the
		// above and below defaults. Using the above for now.
		case "data":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["create"], "users")
			addGroup(org, acl.ACLitems["create"], "clients")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["read"], "users")
			addGroup(org, acl.ACLitems["read"], "clients")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["update"], "users")
			addGroup(org, acl.ACLitems["update"], "clients")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["delete"], "users")
			addGroup(org, acl.ACLitems["delete"], "clients")
			addGroup(org, acl.ACLitems["grant"], "admins")
		case "nodes":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["create"], "users")
			addGroup(org, acl.ACLitems["create"], "clients")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["read"], "users")
			addGroup(org, acl.ACLitems["read"], "clients")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["update"], "users")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["delete"], "users")
			addGroup(org, acl.ACLitems["grant"], "admins")
		case "clients":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["read"], "users")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["delete"], "users")
			addGroup(org, acl.ACLitems["grant"], "admins")
		case "sandboxes":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["create"], "users")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["grant"], "admins")
		default:
			// blank out the previous work
			acl = new(ACL)
		}
	case "groups":
		switch subkind {
		case "admins", "clients", "users":
			for _, perm := range DefaultACLs {
				addGroup(org, acl.ACLitems[perm], "admins")
			}
		case "billing-admins":
			addGroup(org, acl.ACLitems["read"], "billing-admins")
			addGroup(org, acl.ACLitems["update"], "billing-admins")
		default:
			acl = new(ACL)
		}
	}
	return acl
}

func addGroup(org *organization.Organization, aclItem map[string]*ACLitem, name string) util.Gerror {
	g, err := group.Get(org, name)
	if err != nil {
		return err
	}
	aclItem = append(aclItem, g)
}

func Get(org *organization.Organization, kind string) (*ACL, util.Gerror) {

}

func (a *ACL) Save() util.Gerror {

}

func (a *ACL) ToJSON() map[string]interface{} {

}
