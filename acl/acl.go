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
	//"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"log"
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
	Kind       string
	Subkind    string
	ACLitems   map[string]*ACLitem
	Owner      ACLOwner
	Org        *organization.Organization
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

func GetItemACL(org *organization.Organization, item ACLOwner) (*ACL, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	var defacl *ACL
	a, found := ds.Get(org.DataKey("acl-item"), util.JoinStr(item.ContainerKind(), "-", item.ContainerType(), "-", item.GetName()))
	if !found {
		log.Printf("Did not find an ACL for client %s, using default", util.JoinStr(item.ContainerKind(), "-", item.ContainerType(), "-", item.GetName()))
		var err util.Gerror
		defacl, err = defaultACL(org, item.ContainerKind(), item.ContainerType())
		if err != nil {
			return nil, err
		}
		defacl.Owner = item
	} else {
		defacl = a.(*ACL)
	}
	return defacl, nil
}

func (a *ACL) AddGroup(perm string, g *group.Group) util.Gerror {
	if !checkValidPerm(perm) {
		err := util.Errorf("invalid perm %s", perm)
		return err
	}
	if perm == "all" {
		for _, p := range DefaultACLs {
			a.ACLitems[p].Groups = append(a.ACLitems[p].Groups, g)
		}
	} else {
		a.ACLitems[perm].Groups = append(a.ACLitems[perm].Groups, g)
	}
	a.isModified = true
	return nil
}

func (a *ACL) AddActor(perm string, act actor.Actor) util.Gerror {
	if !checkValidPerm(perm) {
		err := util.Errorf("invalid perm %s", perm)
		return err
	}
	if perm == "all" {
		for _, p := range DefaultACLs {
			a.ACLitems[p].Actors = append(a.ACLitems[p].Actors, act)
		}
	} else {
		a.ACLitems[perm].Actors = append(a.ACLitems[perm].Actors, act)
	}
	a.isModified = true
	return nil
}

func (a *ACL) EditFromJSON(perm string, data interface{}) util.Gerror {
	switch data := data.(type) {
	case map[string]interface{}:
		if _, ok := data[perm]; !ok {
			return util.Errorf("acl %s missing from JSON", perm)
		}
		switch aclEdit := data[perm].(type) {
		case map[string]interface{}:
			if actors, ok := aclEdit["actors"].([]interface{}); ok {
				for _, act := range actors {
					switch act := act.(type){
					case string:
						actr, err := actor.GetActor(a.Org, act)
						if err != nil {
							return err
						}
						err = a.AddActor(perm, actr)
						if err != nil {
							return err
						}
					default:
						return util.Errorf("invalid type for actor in acl")
					}
				}
			} else {
				return util.Errorf("invalid acl %s data for actors", perm)
			}
			if groups, ok := aclEdit["groups"].([]interface{}); ok {
				for _, gr := range groups {
					switch gr := gr.(type) {
					case string:
						grp, err := group.Get(a.Org, gr)
						if err != nil {
							return err
						}
						err = a.AddGroup(perm, grp)
						if err != nil {
							return err
						}
					default:
						return util.Errorf("invalid type for group in acl")
					}
				}
			} else {
				return util.Errorf("invalid acl %s data for groups", perm)
			}
		default:
				return util.Errorf("invalid acl %s data", perm)
		}
	default:
		return util.Errorf("invalid acl data")
	}
	return a.Save()
}

func (a *ACL) Save() util.Gerror {
	if config.UsingDB() {

	}
	if a.isModified {
		ds := datastore.New()
		a.isModified = false
		var itemType string
		var key string
		if a.Owner != nil {
			itemType = "acl-item"
			util.JoinStr(a.Owner.ContainerKind(), "-", a.Owner.ContainerType(), "-", a.Owner.GetName())
		} else {
			itemType = "acl"
			key = util.JoinStr(a.Subkind, "-", a.Kind)
		}
		log.Printf("Saving ACL %s :: %s", itemType, key)
		ds.Set(a.Org.DataKey(itemType), key, a)
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

func (a *ACL) CheckPerm(perm string, doer actor.Actor) (bool, util.Gerror) {
	log.Printf("The ACL: %+v", a)
	acli, ok := a.ACLitems[perm]
	log.Printf("The ACLitem: %+v", acli)
	if !ok {
		return false, util.Errorf("invalid perm %s for %s-%s", perm, a.Kind, a.Subkind)
	}
	// first check for user perms in this ACL
	if f, _ := acli.checkForActor(doer); f {
		return f, nil
	}
	for _, g := range acli.Groups {
		if f := g.SeekActor(doer); f {
			return f, nil
		}
	}
	return false, nil
}

func checkValidPerm(perm string) bool {
	for _, p := range DefaultACLs {
		if p == perm {
			return true
		}
	}
	if perm == "all" {
		return true
	}
	return false
}

func (a *ACLitem) checkForActor(actr actor.Actor) (bool, int) {
	for i, ac := range a.Actors {
		log.Printf("ac %s :: actr %s", ac.GetName(), actr.GetName())
		if ac.GetName() == actr.GetName() {
			return true, i
		}
	}
	return false, 0
}

func (a *ACLitem) checkForGroup(g *group.Group) (bool, int) {
	for i, gr := range a.Groups {
		if gr.Name == g.Name {
			return true, i
		}
	}
	return false, 0
}
