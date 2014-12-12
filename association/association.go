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
	"fmt"
	"github.com/ctdk/goiardi/actor"
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
	User    *user.User
	Org     *organization.Organization
	Inviter actor.Actor
}

func (a *AssociationReq) Key() string {
	return util.JoinStr(a.User.Name, "-", a.Org.Name)
}

func (a *Association) Key() string {
	return util.JoinStr(a.User.Name, "-", a.Org.Name)
}

func SetReq(user *user.User, org *organization.Organization, inviter actor.Actor) (*AssociationReq, util.Gerror) {
	if config.UsingDB() {

	}
	assoc := &AssociationReq{user, org, inviter}
	ds := datastore.New()
	_, found := ds.Get("associationreq", assoc.Key())
	if found {
		err := util.Errorf("The invite already exists.")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	_, found = ds.Get("association", assoc.Key())
	if found {
		err := util.Errorf("The association already exists.")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	ds.Set("associationreq", assoc.Key(), assoc)
	ds.SetAssociationReq(org.Name, "users", user.Name, user)
	ds.SetAssociationReq(user.Name, "organizations", org.Name, org)
	return assoc, nil
}

func GetReq(key string) (*AssociationReq, util.Gerror) {
	var assoc *AssociationReq
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		a, found := ds.Get("associationreq", key)
		if !found {
			gerr := util.Errorf("Cannot find association request: %s", key)
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
	// This one I think *does* happen. I think.
	_, err := SetAssoc(a.User, a.Org)
	if err != nil {
		return err
	}
	g, err := group.Get(a.Org, "users")
	if err != nil {
		return err
	}
	err = g.AddActor(a.User)
	if err != nil {
		return err
	}
	// apparently we create a USAG, but what are they like?
	// use BS hex value until we have some idea what's supposed to be there
	usagName := fmt.Sprintf("%x", []byte(a.User.Name))
	usag, err := group.New(a.Org, usagName)
	if err != nil {
		return err
	}
	err = usag.Save()
	if err != nil {
		return nil
	}
	err = g.AddGroup(usag)
	if err != nil {
		return err
	}
	err = g.Save()
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
	ds.Delete("associationreq", a.Key())
	ds.DelAssociationReq(a.Org.Name, "users", a.User.Name)
	ds.DelAssociationReq(a.User.Name, "organizations", a.Org.Name)
	return nil
}

func OrgAssocReqs(user *user.User) ([]*organization.Organization, util.Gerror) {
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

func OrgAssociations(user *user.User) ([]*organization.Organization, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	o := ds.GetAssociations(user.Name, "organizations")
	orgs := make([]*organization.Organization, len(o))
	for i, v := range o {
		orgs[i] = v.(*organization.Organization)
	}
	return orgs, nil
}

func OrgsAssociationReqCount(user *user.User) (int, util.Gerror) {
	if config.UsingDB() {

	}
	orgs, err := OrgAssocReqs(user)
	if err != nil {
		return 0, err
	}
	count := len(orgs)
	return count, nil
}

func UsersAssociationReqCount(org *organization.Organization) (int, util.Gerror) {
	if config.UsingDB() {

	}
	users, err := UserAssocReqs(org)
	if err != nil {
		return 0, err
	}
	count := len(users)
	return count, nil
}

func UserAssocReqs(org *organization.Organization) ([]*user.User, util.Gerror) {
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

func UserAssociations(org *organization.Organization) ([]*user.User, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	u := ds.GetAssociations(org.Name, "users")
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
	orgs, err := OrgAssocReqs(user)
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
	users, err := UserAssocReqs(org)
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

func DelAllUserAssociations(user *user.User) util.Gerror {
	// these two will be vastly easier with the db, eh.
	if config.UsingDB() {

	}
	orgs, err := OrgAssociations(user)
	if err != nil {
		return err
	}
	for _, o := range orgs {
		a, err := GetAssoc(user, o)
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

func DelAllOrgAssociations(org *organization.Organization) util.Gerror {
	if config.UsingDB() {

	}
	users, err := UserAssociations(org)
	if err != nil {
		return err
	}
	for _, u := range users {
		a, err := GetAssoc(u, org)
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
	orgs, err := OrgAssocReqs(user)
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
	users, err := UserAssocReqs(org)
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

func SetAssoc(user *user.User, org *organization.Organization) (*Association, util.Gerror) {
	assoc := &Association{user, org}
	ds := datastore.New()
	_, found := ds.Get("association", assoc.Key())
	if found {
		err := util.Errorf("User %s already associated with org %s", user.Name, org.Name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	ds.Set("association", assoc.Key(), assoc)
	ds.SetAssociation(org.Name, "users", user.Name, user)
	ds.SetAssociation(user.Name, "organizations", org.Name, org)
	return assoc, nil
}

func GetAssoc(user *user.User, org *organization.Organization) (*Association, util.Gerror) {
	var assoc *Association
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		key := util.JoinStr(user.Name, "-", org.Name)
		a, found := ds.Get("association", key)
		if !found {
			gerr := util.Errorf("'%s' not associated with organization '%s'", user.Name, org.Name)
			gerr.SetStatus(http.StatusForbidden)
			return nil, gerr
		}
		if a != nil {
			assoc = a.(*Association)
		}
	}
	return assoc, nil
}

func (a *Association) Delete() util.Gerror {
	if config.UsingDB() {

	}
	ds := datastore.New()
	usagName := fmt.Sprintf("%x", []byte(a.User.Name))
	usag, err := group.Get(a.Org, usagName)
	if err != nil {
		return err
	}
	allGroups := group.AllGroups(a.Org)
	for _, g := range allGroups {
		if g.Name == usag.Name {
			continue
		}
		g.DelGroup(usag)
		g.DelActor(a.User)
		err = g.Save()
		if err != nil {
			return err
		}
	}

	err = usag.Delete()
	if err != nil {
		return err
	}

	ds.Delete("association", a.Key())
	ds.DelAssociation(a.Org.Name, "users", a.User.Name)
	ds.DelAssociation(a.User.Name, "organizations", a.Org.Name)
	return nil
}
