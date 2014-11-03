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
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

type Association struct {
	User *user.User
	Org  *organization.Organization
}

type AssociationReq struct {
	User *user.User
	Org  *organization.Organization
}

func (a *AssociationReq) Key() string {
	return util.JoinStr(a.User.Name, "-", a.Org.Name)
}

func SetReq(user *user.User, org *organization.Organization) (*AssociationReq, util.Gerror) {
	if config.UsingDB() {

	}
	assoc := &AssociationReq{user, org}
	ds := datastore.New()
	_, found := ds.Get("association", assoc.Key())
	if found {
		err := util.Errorf("assocation %s already exists", assoc.Key())
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	ds.Set("association", assoc.Key(), assoc)
	ds.SetAssociationReq(org.Name, "users", user.Name, user)
	ds.SetAssociationReq(user.Name, "organizations", org.Name, org)
	return assoc, nil
}

func GetReq(key string) (*AssociationReq, util.Gerror) {
	var assoc *AssociationReq
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		a, found := ds.Get("association", key)
		if !found {
			gerr := util.Errorf("Association %s not found", key)
			gerr.SetStatus(http.StatusNotFound)
			return nil, gerr
		}
		if a != nil {
			assoc = a.(*AssociationReq)
		}
	}
	return assoc, nil
}

func (a *AssociationReq) Accept() util.Gerror {
	if config.UsingDB() {

	}
	// group stuff happens here, once that all gets figured out
	g, err := group.Get(a.Org, "users")
	if err != nil {
		return err
	}
	err = g.AddActor(a.User)
	if err != nil {
		return err
	}
	return a.Delete()
}

func (a *AssociationReq) Reject() util.Gerror {
	return a.Delete()
}

func (a *AssociationReq) Delete() util.Gerror {
	if config.UsingDB() {

	}
	ds := datastore.New()
	ds.Delete("association", a.Key())
	ds.DelAssociationReq(a.Org.Name, "users", a.User.Name)
	ds.DelAssociationReq(a.User.Name, "organizations", a.Org.Name)
	return nil
}

func Orgs(user *user.User) ([]*organization.Organization, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	o := ds.GetAssociationReqs(user.Name, "organizations")
	orgs := make([]*organization.Organization, len(o))
	for i, v := range o {
		orgs[i] = v.(*organization.Organization)
	}
	return orgs, nil
}

func OrgsAssociationReqCount(user *user.User) (int, util.Gerror) {
	if config.UsingDB() {

	}
	orgs, err := Orgs(user)
	if err != nil {
		return 0, err
	}
	count := len(orgs)
	return count, nil
}

func UsersAssociationReqCount(org *organization.Organization) (int, util.Gerror) {
	if config.UsingDB() {

	}
	users, err := Users(org)
	if err != nil {
		return 0, err
	}
	count := len(users)
	return count, nil
}

func Users(org *organization.Organization) ([]*user.User, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	u := ds.GetAssociationReqs(org.Name, "users")
	users := make([]*user.User, len(u))
	for i, v := range u {
		users[i] = v.(*user.User)
	}
	return users, nil
}

func DelAllUserAssocReqs(user *user.User) util.Gerror {
	// these two will be vastly easier with the db, eh.
	if config.UsingDB() {

	}
	orgs, err := Orgs(user)
	if err != nil {
		return err
	}
	for _, o := range orgs {
		key := util.JoinStr(user.Name, "-", o.Name)
		a, err := GetReq(key)
		if err != nil {
			return err
		}
		err = a.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

func DelAllOrgAssocReqs(org *organization.Organization) util.Gerror {
	if config.UsingDB() {

	}
	users, err := Users(org)
	if err != nil {
		return err
	}
	for _, u := range users {
		key := util.JoinStr(u.Name, "-", org.Name)
		a, err := GetReq(key)
		if err != nil {
			return err
		}
		err = a.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

func GetAllOrgsAssociationReqs(user *user.User) ([]*AssociationReq, util.Gerror) {
	if config.UsingDB() {

	}
	orgs, err := Orgs(user)
	if err != nil {
		return nil, err
	}
	assoc := make([]*AssociationReq, len(orgs))
	for i, o := range orgs {
		key := util.JoinStr(user.Name, "-", o.Name)
		a, err := GetReq(key)
		if err != nil {
			return nil, err
		}
		assoc[i] = a
	}
	return assoc, nil
}

func GetAllUsersAssociationReqs(org *organization.Organization) ([]*AssociationReq, util.Gerror) {
	if config.UsingDB() {

	}
	users, err := Users(org)
	if err != nil {
		return nil, err
	}
	assoc := make([]*AssociationReq, len(users))
	for i, u := range users {
		key := util.JoinStr(u.Name, "-", org.Name)
		a, err := GetReq(key)
		if err != nil {
			return nil, err
		}
		assoc[i] = a
	}
	return assoc, nil
}
