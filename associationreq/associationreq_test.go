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

package associationreq

import (
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"testing"
)

func TestAssociationCreation(t *testing.T) {
	gob.Register(new(AssociationReq))
	gob.Register(new(organization.Organization))
	gob.Register(new(user.User))
	gob.Register(make(map[string]interface{}))
	u, _ := user.New("user1")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	o, _ := organization.New("org", "org-porg")
	o.Save()
	assoc, err := Set(u, o)
	if err != nil {
		t.Errorf(err.Error())
	}
	a2, err := Get(assoc.Key())
	if err != nil {
		t.Errorf(err.Error())
	}
	if a2.Key() != assoc.Key() {
		t.Errorf("association keys should have matched, got %s and %s", a2.Key(), assoc.Key())
	}
}

func TestAssociationDeletion(t *testing.T) {
	u, _ := user.New("user2")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	o, _ := organization.New("org2", "org-porg")
	o.Save()
	assoc, err := Set(u, o)
	if err != nil {
		t.Errorf(err.Error())
	}
	key := assoc.Key()
	assoc.Delete()
	a2, err := Get(key)
	if err == nil {
		t.Errorf("deleting %s didn't work. Value: %v", key, a2)
	}
}

func TestOrgListing(t *testing.T) {
	u, _ := user.New("user3")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	for n := 0; n < 5; n++ {
		name := fmt.Sprintf("orglist%d", n)
		o, e := organization.New(name, fmt.Sprintf("%s org thing", name))
		if e != nil {
			t.Errorf(e.Error())
		}
		o.Save()
		_, err := Set(u, o)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
	orgs, err := Orgs(u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(orgs) != 5 {
		t.Errorf("the number of orgs associated with the user should have been 5, got %d", len(orgs))
	}
}

func TestUserListing(t *testing.T) {
	o, _ := organization.New("userlist", "user list org")
	o.Save()
	pass := "123456"
	for n := 0; n < 5; n++ {
		name := fmt.Sprintf("userlist%d", n)
		u, _ := user.New(name)
		u.SetPasswd(pass)
		u.Save()
		_, err := Set(u, o)
		if err != nil {
			t.Errorf(err.Error())
		}
	}
	users, err := Users(o)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(users) != 5 {
		t.Errorf("the number of users associated with the org should have been 5, got %d", len(users))
	}
}

func TestDelUserAssoc(t *testing.T) {
	o, _ := organization.Get("userlist")
	err := DelAllOrgAssoc(o)
	if err != nil {
		t.Errorf(err.Error())
	}
	users, err := Users(o)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(users) != 0 {
		t.Errorf("user associations for this org should have been 0, got %d", len(users))
	}
}

func TestDelOrgAssoc(t *testing.T) {
	u, _ := user.Get("user3")
	err := DelAllUserAssoc(u)
	if err != nil {
		t.Errorf(err.Error())
	}
	orgs, err := Orgs(u)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(orgs) != 0 {
		t.Errorf("org associations for this user should have been 0, got %d", len(orgs))
	}
}
