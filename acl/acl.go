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
	"github.com/ctdk/goiardi/association"
	//"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"log"
	"sync"
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
	m      sync.RWMutex
}

type ACL struct {
	Kind       string
	Subkind    string
	ACLitems   map[string]*ACLitem
	Owner      ACLOwner
	Org        *organization.Organization
	isModified bool
	m      sync.RWMutex
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
			ggerr := acl.addActor(perm, defUser)
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
		case "log-infos":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["create"], "users")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["grant"], "admins")
		case "reports":
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["create"], "users")
			addGroup(org, acl.ACLitems["create"], "clients")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["delete"], "admins")
			addGroup(org, acl.ACLitems["grant"], "admins")
		case "shoveys": // certain to be modified further later
			addGroup(org, acl.ACLitems["create"], "admins")
			addGroup(org, acl.ACLitems["read"], "admins")
			addGroup(org, acl.ACLitems["update"], "admins")
			addGroup(org, acl.ACLitems["update"], "clients")
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
				ggerr = acl.addActor(perm, defUser)
				if ggerr != nil {
					return nil, ggerr
				}
			}
		case "billing-admins":
			addGroup(org, acl.ACLitems["read"], "billing-admins")
			addGroup(org, acl.ACLitems["update"], "billing-admins")
			for _, perm := range DefaultACLs {
				ggerr := acl.addActor(perm, defUser)
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
		//defacl, err = defaultACL(org, item.ContainerKind(), item.ContainerType())
		// Experiment: inherit the parent container's ACL, rather than
		// the default for this type.
		defacl, err = Get(org, item.ContainerKind(), item.ContainerType())
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
	a.m.Lock()
	defer a.m.Unlock()
	return a.addGroup(perm, g)
}

func (a *ACL) AddActor(perm string, act actor.Actor) util.Gerror {
	a.m.Lock()
	defer a.m.Unlock()
	return a.addActor(perm, act)
}

func (a *ACL) addGroup(perm string, g *group.Group) util.Gerror {
	if !checkValidPerm(perm) {
		err := util.Errorf("invalid perm %s", perm)
		return err
	}
	if perm == "all" {
		for _, p := range DefaultACLs {
			a.ACLitems[p].m.Lock()
			if f, _ := a.ACLitems[p].checkForGroup(g); !f {
				a.ACLitems[p].Groups = append(a.ACLitems[p].Groups, g)
			}
			a.ACLitems[p].m.Unlock()
		}
	} else {
		a.ACLitems[perm].m.Lock()
		if f, _ := a.ACLitems[perm].checkForGroup(g); !f {
			a.ACLitems[perm].Groups = append(a.ACLitems[perm].Groups, g)
		}
		a.ACLitems[perm].m.Unlock()
	}
	a.isModified = true
	return nil
}

func (a *ACL) addActor(perm string, act actor.Actor) util.Gerror {
	if !checkValidPerm(perm) {
		err := util.Errorf("invalid perm %s", perm)
		return err
	}
	if perm == "all" {
		for _, p := range DefaultACLs {
			a.ACLitems[p].m.Lock()
			if f, _ := a.ACLitems[p].checkForActor(act); !f {
				a.ACLitems[p].Actors = append(a.ACLitems[p].Actors, act)
			}
			a.ACLitems[p].m.Unlock()
		}
	} else {
		a.ACLitems[perm].m.Lock()
		if f, _ := a.ACLitems[perm].checkForActor(act); !f {
			a.ACLitems[perm].Actors = append(a.ACLitems[perm].Actors, act)
		}
		a.ACLitems[perm].m.Unlock()
	}
	a.isModified = true
	return nil
}

func (a *ACL) EditFromJSON(perm string, data interface{}) util.Gerror {
	a.m.Lock()
	defer a.m.Unlock()
	switch data := data.(type) {
	case map[string]interface{}:
		if _, ok := data[perm]; !ok {
			return util.Errorf("acl %s missing from JSON", perm)
		}
		switch aclEdit := data[perm].(type) {
		case map[string]interface{}:
			a.ACLitems[perm].m.Lock()
			defer a.ACLitems[perm].m.Unlock()
			var acts []actor.Actor
			var gs []*group.Group
			if actors, ok := aclEdit["actors"].([]interface{}); ok {
				acts = make([]actor.Actor, 0, len(actors))
				for _, act := range actors {
					switch act := act.(type){
					case string:
						actr, err := actor.GetActor(a.Org, act)
						if err != nil {
							return err
						}
						acts = append(acts, actr)
					default:
						return util.Errorf("invalid type for actor in acl")
					}
				}
			} else {
				return util.Errorf("invalid acl %s data for actors", perm)
			}
			if groups, ok := aclEdit["groups"].([]interface{}); ok {
				gs = make([]*group.Group, 0, len(groups))
				for _, gr := range groups {
					switch gr := gr.(type) {
					case string:
						grp, err := group.Get(a.Org, gr)
						if err != nil {
							return err
						}
						gs = append(gs, grp)
					default:
						return util.Errorf("invalid type for group in acl")
					}
				}
			} else {
				return util.Errorf("invalid acl %s data for groups", perm)
			}
			a.ACLitems[perm].Actors = acts
			a.ACLitems[perm].Groups = gs
		default:
				return util.Errorf("invalid acl %s data", perm)
		}
	default:
		return util.Errorf("invalid acl data")
	}
	a.isModified = true
	return a.save()
}

func (a *ACL) Save() util.Gerror {
	a.m.Lock()
	defer a.m.Unlock()
	return a.save()
}

func (a *ACL) save() util.Gerror {
	if config.UsingDB() {

	}
	if a.isModified {
		ds := datastore.New()
		a.isModified = false
		var itemType string
		var key string
		if a.Owner != nil {
			itemType = "acl-item"
			key = util.JoinStr(a.Owner.ContainerKind(), "-", a.Owner.ContainerType(), "-", a.Owner.GetName())
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
	a.m.RLock()
	defer a.m.RUnlock()
	aclJSON := make(map[string]interface{})
	for k, v := range a.ACLitems {
		aclJSON[k] = v.ToJSON()
	}
	return aclJSON
}

func (acli *ACLitem) ToJSON() map[string]interface{} {
	acli.m.RLock()
	defer acli.m.RUnlock()
	r := make(map[string]interface{}, 2)
	ractors := make([]string, len(acli.Actors))
	rgroups := make([]string, len(acli.Groups))
	for i, act := range acli.Actors {
		ractors[i] = act.GetName()
	}
	for i, gr := range acli.Groups {
		rgroups[i] = gr.Name
	}
	r["actors"] = ractors
	r["groups"] = rgroups
	return r
}

func (a *ACL) CheckPerm(perm string, doer actor.Actor) (bool, util.Gerror) {
	a.m.RLock()
	defer a.m.RUnlock()
	log.Printf("The ACL: %+v", a)
	acli, ok := a.ACLitems[perm]
	acli.m.RLock()
	defer acli.m.RUnlock()

	log.Printf("The ACLitem: %+v", acli)
	if !ok {
		return false, util.Errorf("invalid perm %s for %s-%s", perm, a.Kind, a.Subkind)
	}
	// check for user perms in this ACL
	if f, _ := acli.checkForActor(doer); f {
		return f, nil
	}
	for _, g := range acli.Groups {
		if f := g.SeekActor(doer); f {
			return f, nil
		}
	}
	if doer.IsUser() {
		_, err := association.GetAssoc(doer.(*user.User), a.Org)
		if err != nil {
			return false, err
		}
	} else {
		if doer.OrgName() != a.Org.Name {
			err := util.Errorf("client %s is not associated iwth org %s", doer.GetName(), a.Org.Name)
			err.SetStatus(http.StatusForbidden)
			return false, err
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
	a.m.RLock()
	defer a.m.RUnlock()
	for i, gr := range a.Groups {
		if gr.Name == g.Name {
			return true, i
		}
	}
	return false, 0
}
