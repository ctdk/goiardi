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
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
)

var DefaultACLs = [5]string{
	"create",
	"read",
	"update",
	"delete",
	"grant",
}

var DefaultUser = "pivotal" // should this be configurable?

type ACLOwner interface {
	GetName() string
	ContainerKind() string
	ContainerType() string
}

type ACLitem struct {
	Perm   string
	Actors []actor.Actor
	Groups []*group.Group
}

type ACL struct {
	Kind string
	Subkind string
	ACLitems map[string]*ACLitem
	Owner    ACLOwner
	Org *organization.Organization
	isModified bool
}

func defaultACL(org *organization.Organization, kind string, subkind string) (*ACL, util.Gerror) {
	acl := new(ACL)
	acl.Kind = kind
	acl.Subkind = subkind
	acl.Org = org
	// almost always we'd want these default acls
	acl.ACLitems = make(map[string]*ACLitem)
	for _, a := range DefaultACLs {
		acl.ACLitems[a] = &ACLitem{Perm: a}
	}
	defUser, err := user.Get(DefaultUser)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "containers":
		// by default, all of these seem to have the same default
		// user
		for _, perm := range DefaultACLs {
			ggerr := acl.AddActor(perm, defUser)
			if ggerr != nil {
				return nil, ggerr
			}
		}
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
			acl.ACLitems = nil
		}
	case "groups":
		switch subkind {
		case "admins", "clients", "users":
			for _, perm := range DefaultACLs {
				ggerr := addGroup(org, acl.ACLitems[perm], "admins")
				if ggerr != nil {
					return nil, ggerr
				}
				ggerr = acl.AddActor(perm, defUser)
				if ggerr != nil {
					return nil, ggerr
				}
			}
		case "billing-admins":
			addGroup(org, acl.ACLitems["read"], "billing-admins")
			addGroup(org, acl.ACLitems["update"], "billing-admins")
			for _, perm := range DefaultACLs {
				ggerr := acl.AddActor(perm, defUser)
				if ggerr != nil {
					return nil, ggerr
				}
			}
		default:
			acl.ACLitems = nil
		}
	default:
		e := util.Errorf("Ok got to default with kind %s, subkind %s", kind, subkind)
		return nil, e
	}
	return acl, nil
}

func addGroup(org *organization.Organization, aclItem *ACLitem, name string) util.Gerror {
	g, err := group.Get(org, name)
	if err != nil {
		return err
	}
	aclItem.Groups = append(aclItem.Groups, g)
	return nil
}

func Get(org *organization.Organization, kind string, subkind string) (*ACL, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	a, found := ds.Get(org.DataKey("acl"), util.JoinStr(kind, "-", subkind))
	if !found {
		return defaultACL(org, kind, subkind)
	}
	return a.(*ACL), nil
}

func (a *ACL) AddGroup(perm string, g *group.Group) util.Gerror {
	if !checkValidPerm(perm){
		err := util.Errorf("invalid perm %s", perm)
		return err
	}
	a.ACLitems[perm].Groups = append(a.ACLitems[perm].Groups, g)
	return nil
}

func (a *ACL) AddActor(perm string, act actor.Actor) util.Gerror {
	if !checkValidPerm(perm){
		err := util.Errorf("invalid perm %s", perm)
		return err
	}
	a.ACLitems[perm].Actors = append(a.ACLitems[perm].Actors, act)
	return nil
}

func (a *ACL) Save() util.Gerror {
	if config.UsingDB() {

	}
	if a.isModified {
		ds := datastore.New()
		ds.Set(a.Org.DataKey("acl"), util.JoinStr(a.Subkind, "-", a.Kind), a)
	}
	return nil
}

func (a *ACL) ToJSON() map[string]interface{} {
	aclJSON := make(map[string]interface{})
	for k, v := range a.ACLitems {
		r := make(map[string][]string, 2)
		r["actors"] = make([]string, len(v.Actors))
		r["groups"] = make([]string, len(v.Groups))
		for i, act := range v.Actors {
			r["actors"][i] = act.GetName()
		}
		for i, gr := range v.Groups {
			r["groups"][i] = gr.Name
		}
		aclJSON[k] = r
	}
	return aclJSON
}

func checkValidPerm(perm string) bool {
	for _, p := range DefaultACLs {
		if p == perm {
			return true
		}
	}
	return false
}
