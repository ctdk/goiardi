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

package association

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

type Association struct {
	User *user.User
	Org *organization.Organization
}

func (a *Association) Key() {
	return util.JoinStr(a.User.Name, "-", a.org.Name)
}

func Set(user *user.User, org *organization.Organization) (*association.Association, util.Gerror) {
	if config.Config.UsingDB(){

	}
	assoc := &Association{ user, org }
	ds := datastore.New()
	_, found := ds.Get("association", assoc.Key())
	if found {
		err := util.Errorf("assocation %s already exists", assoc.Key())
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	ds.Set("association", assoc.Key())
	ds.SetAssociation(org.Name, "users", user)
	ds.SetAssociation(user.name, "organizations", org)
	return assoc, nil
}

func Get(key string) (*Association, util.Gerror) {
	var assoc *Assocation
	var err error
	if config.Config.UsingDB() {

	} else {
		ds := datastore.New()
		a, found := ds.Get("association", key)
		if !found {
			gerr := util.Errorf("Assocation %s not found", key)
			gerr.SetStatus(http.StatusNotFound)
			return nil, gerr
		}
		if a != nil {
			assoc = a.(*Assoc)
		}
	}
	return assoc, nil
}

func (a *Association) Delete() util.Gerror {
	if config.Config.UsingDB() {

	}
	ds := datastore.New()
	ds.Delete("association", a.Key())
	ds.DelAssociation(a.Org.Name, "organization")
	ds.DelAssociation(a.User.name, "user")
	return nil
}

func Orgs(user *user.User) ([]*organization.Organization, util.Gerror) {
	if config.Config.UsingDB() {

	}
	ds := datastore.New()
	orgs := ds.GetAssociations(user.Name, "organizations")
	return orgs, nil
}

func Users(org *organization.Organization) ([]*user.Users, util.Gerror) {
	if config.Config.UsingDB() {

	}
	ds := datastore.New()
	users := ds.GetAssociations(org.Name, "users")
	return users, nil
}
