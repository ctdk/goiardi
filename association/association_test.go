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

package association

import (
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/fakeacl"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"testing"
)

var pivotal *user.User

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
}

func TestAssociationReqCreation(t *testing.T) {
	gob.Register(new(AssociationReq))
	gob.Register(new(organization.Organization))
	gob.Register(new(user.User))
	gob.Register(make(map[string]interface{}))
	gob.Register(new(Association))
	gob.Register(new(group.Group))
	u, _ := user.New("user1")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	up, _ := user.New("pivotal")
	up.SetPasswd(pass)
	up.Save()
	pivotal = up
	o, _ := organization.New("org", "org-porg")
	fakeacl.LoadFakeACL(o)
	o.Save()
	assoc, err := SetReq(u, o, pivotal)
	if err != nil {
		t.Error(err.Error())
	}
	a2, err := GetReq(u, o)
	if err != nil {
		t.Error(err.Error())
	}
	if a2.Key() != assoc.Key() {
		t.Errorf("association keys should have matched, got %s and %s", a2.Key(), assoc.Key())
	}
}

func TestAssociationReqDeletion(t *testing.T) {
	u, _ := user.New("user2")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	o, _ := organization.New("org2", "org-porg")
	fakeacl.LoadFakeACL(o)
	o.Save()
	assoc, err := SetReq(u, o, pivotal)
	if err != nil {
		t.Error(err.Error())
	}
	key := assoc.Key()
	assoc.Delete()
	a2, err := GetReq(u, o)
	if err == nil {
		t.Errorf("deleting %s didn't work. Value: %v", key, a2)
	}
}

func TestAcceptance(t *testing.T) {
	u, _ := user.New("user100")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()

	o, _ := organization.New("org100", "org-porg")
	fakeacl.LoadFakeACL(o)
	o.Save()
	err := group.MakeDefaultGroups(o)
	if err != nil {
		t.Error(err.Error())
	}
	assoc, err := SetReq(u, o, pivotal)
	if err != nil {
		t.Error(err.Error())
	}
	err = assoc.Accept()
	if err != nil {
		t.Error(err.Error())
	}
	_, err = GetAssoc(u, o)
	if err != nil {
		t.Error(err.Error())
	}
	u2, _ := user.New("user101")
	u2.SetPasswd(pass)
	u2.Save()
	_, err = GetAssoc(u2, o)
	if err == nil {
		t.Error("found association when there should not have been one")
	}
}

func TestAcceptRemoveReq(t *testing.T) {
	u, _ := user.New("user103")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	o, _ := organization.New("org103", "org-porg")
	fakeacl.LoadFakeACL(o)
	o.Save()
	err := group.MakeDefaultGroups(o)
	if err != nil {
		t.Error(err.Error())
	}
	assoc, err := SetReq(u, o, pivotal)
	if err != nil {
		t.Error(err.Error())
	}
	err = assoc.Accept()
	if err != nil {
		t.Error(err.Error())
	}
	areq, _ := GetReq(u, o)
	if areq != nil {
		t.Errorf("Curious, this req shouldn't have been there: %+v", areq)
	}
}

func TestOrgReqListing(t *testing.T) {
	u, _ := user.New("user3")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	for n := 0; n < 5; n++ {
		name := fmt.Sprintf("orglist%d", n)
		o, e := organization.New(name, fmt.Sprintf("%s org thing", name))
		if e != nil {
			t.Error(e.Error())
		}
		fakeacl.LoadFakeACL(o)
		o.Save()
		_, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
	}
	orgs, err := OrgAssocReqs(u)
	if err != nil {
		t.Error(err.Error())
	}
	if len(orgs) != 5 {
		t.Errorf("the number of orgs associated with the user should have been 5, got %d", len(orgs))
	}
}

func TestUserReqListing(t *testing.T) {
	o, _ := organization.New("userlist", "user list org")
	fakeacl.LoadFakeACL(o)
	o.Save()
	pass := "123456"
	for n := 0; n < 5; n++ {
		name := fmt.Sprintf("userlist%d", n)
		u, _ := user.New(name)
		u.SetPasswd(pass)
		u.Save()
		_, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
	}
	users, err := userAssocReqs(o)
	if err != nil {
		t.Error(err.Error())
	}
	if len(users) != 5 {
		t.Errorf("the number of users associated with the org should have been 5, got %d", len(users))
	}
}

func TestUserAssocListing(t *testing.T) {
	o, _ := organization.New("userlistz", "user list org")
	fakeacl.LoadFakeACL(o)
	o.Save()
	group.MakeDefaultGroups(o)
	pass := "123456"
	for n := 0; n < 5; n++ {
		name := fmt.Sprintf("userlistz%d", n)
		u, err := user.New(name)
		if err != nil {
			t.Error(err.Error())
		}
		u.SetPasswd(pass)
		u.Save()
		r, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
		r.Accept()
	}
	users, err := UserAssociations(o)
	if err != nil {
		t.Error(err.Error())
	}
	if len(users) != 5 {
		t.Errorf("the number of users associated with the org should have been 5, got %d", len(users))
	}
}

func TestOrgAssocListing(t *testing.T) {
	u, _ := user.New("user300")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	for n := 0; n < 5; n++ {
		name := fmt.Sprintf("orglistA%d", n)
		o, e := organization.New(name, fmt.Sprintf("%s org thing", name))
		if e != nil {
			t.Error(e.Error())
		}
		fakeacl.LoadFakeACL(o)
		o.Save()
		group.MakeDefaultGroups(o)
		r, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
		r.Accept()
	}
	orgs, err := OrgAssociations(u)
	if err != nil {
		t.Error(err.Error())
	}
	if len(orgs) != 5 {
		t.Errorf("the number of orgs associated with the user should have been 5, got %d", len(orgs))
	}
}

func TestDelUserAssocReq(t *testing.T) {
	o, _ := organization.Get("userlist")
	err := DelAllOrgAssocReqs(o)
	if err != nil {
		t.Error(err.Error())
	}
	users, err := userAssocReqs(o)
	if err != nil {
		t.Error(err.Error())
	}
	if len(users) != 0 {
		t.Errorf("user associations for this org should have been 0, got %d", len(users))
	}
}

func TestDelOrgAssocReq(t *testing.T) {
	u, _ := user.Get("user3")
	err := DelAllUserAssocReqs(u)
	if err != nil {
		t.Error(err.Error())
	}
	orgs, err := OrgAssocReqs(u)
	if err != nil {
		t.Error(err.Error())
	}
	if len(orgs) != 0 {
		t.Errorf("org associations for this user should have been 0, got %d", len(orgs))
	}
}

func TestDelOneUserOrgAssociation(t *testing.T) {
	u, _ := user.New("user301")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	o, _ := organization.New("userlistz1", "user list org")
	fakeacl.LoadFakeACL(o)
	o.Save()
	group.MakeDefaultGroups(o)
	r, err := SetReq(u, o, pivotal)
	if err != nil {
		t.Error(err.Error())
	}
	r.Accept()
	assoc, err := GetAssoc(u, o)
	if err != nil {
		t.Error(err.Error())
	}
	err = assoc.Delete()
	if err != nil {
		t.Error(err.Error())
	}
	a2, _ := GetAssoc(u, o)
	if a2 != nil {
		t.Error("association still found")
	}
	ol, err := UserAssociations(o)
	if err != nil {
		t.Error(err.Error())
	}
	if len(ol) != 0 {
		t.Error("Found user associations with org, but shouldn't have")
	}
	ul, err := OrgAssociations(u)
	if err != nil {
		t.Error(err.Error())
	}
	if len(ul) != 0 {
		t.Error("Found org associations with user, but shouldn't have")
	}
}

func TestGetAllOrgsAssociationReqs(t *testing.T) {
	u, _ := user.New("user-assoc-reqs")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	for n := 0; n < 3; n++ {
		name := fmt.Sprintf("org-assoc-req-%d", n)
		o, _ := organization.New(name, fmt.Sprintf("%s org", name))
		fakeacl.LoadFakeACL(o)
		o.Save()
		_, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
	}
	reqs, err := GetAllOrgsAssociationReqs(u)
	if err != nil {
		t.Fatalf("GetAllOrgsAssociationReqs failed: %s", err.Error())
	}
	if len(reqs) != 3 {
		t.Errorf("expected 3 association reqs, got %d", len(reqs))
	}
}

func TestGetAllUsersAssociationReqs(t *testing.T) {
	o, _ := organization.New("user-assoc-reqs-org", "users assoc req org")
	fakeacl.LoadFakeACL(o)
	o.Save()
	pass := "123456"
	for n := 0; n < 3; n++ {
		name := fmt.Sprintf("user-assoc-req-%d", n)
		u, _ := user.New(name)
		u.SetPasswd(pass)
		u.Save()
		_, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
	}
	reqs, err := GetAllUsersAssociationReqs(o)
	if err != nil {
		t.Fatalf("GetAllUsersAssociationReqs failed: %s", err.Error())
	}
	if len(reqs) != 3 {
		t.Errorf("expected 3 association reqs, got %d", len(reqs))
	}
}

func TestDelAllUserAssociations(t *testing.T) {
	u, _ := user.New("user-cascade-assoc")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	for n := 0; n < 3; n++ {
		name := fmt.Sprintf("org-cascade-assoc-%d", n)
		o, _ := organization.New(name, fmt.Sprintf("%s org", name))
		fakeacl.LoadFakeACL(o)
		o.Save()
		if err := group.MakeDefaultGroups(o); err != nil {
			t.Error(err.Error())
		}
		r, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
		if err := r.Accept(); err != nil {
			t.Error(err.Error())
		}
	}
	if err := DelAllUserAssociations(u); err != nil {
		t.Fatalf("DelAllUserAssociations failed: %s", err.Error())
	}
	orgs, err := OrgAssociations(u)
	if err != nil {
		t.Error(err.Error())
	}
	if len(orgs) != 0 {
		t.Errorf("expected 0 org associations, got %d", len(orgs))
	}
}

func TestDelAllOrgAssociations(t *testing.T) {
	o, _ := organization.New("org-cascade-assoc", "org cascade assoc")
	fakeacl.LoadFakeACL(o)
	o.Save()
	if err := group.MakeDefaultGroups(o); err != nil {
		t.Error(err.Error())
	}
	pass := "123456"
	for n := 0; n < 3; n++ {
		name := fmt.Sprintf("user-cascade-assoc-%d", n)
		u, _ := user.New(name)
		u.SetPasswd(pass)
		u.Save()
		r, err := SetReq(u, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
		if err := r.Accept(); err != nil {
			t.Error(err.Error())
		}
	}
	if err := DelAllOrgAssociations(o); err != nil {
		t.Fatalf("DelAllOrgAssociations failed: %s", err.Error())
	}
	users, err := UserAssociations(o)
	if err != nil {
		t.Error(err.Error())
	}
	if len(users) != 0 {
		t.Errorf("expected 0 user associations, got %d", len(users))
	}
}

func TestAssociationReqReject(t *testing.T) {
	u, _ := user.New("user-reject")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	o, _ := organization.New("org-reject", "reject org")
	fakeacl.LoadFakeACL(o)
	o.Save()
	req, err := SetReq(u, o, pivotal)
	if err != nil {
		t.Fatalf("SetReq failed: %s", err.Error())
	}
	if err := req.Reject(); err != nil {
		t.Fatalf("Reject failed: %s", err.Error())
	}
	if r, err := GetReq(u, o); err == nil {
		t.Errorf("expected req to be deleted after reject, got %+v", r)
	}
}

func TestAssociationReqCounts(t *testing.T) {
	u, _ := user.New("user-counts")
	pass := "123456"
	u.SetPasswd(pass)
	u.Save()
	o, _ := organization.New("org-counts", "counts org")
	fakeacl.LoadFakeACL(o)
	o.Save()
	for n := 0; n < 4; n++ {
		name := fmt.Sprintf("user-count-%d", n)
		uu, _ := user.New(name)
		uu.SetPasswd(pass)
		uu.Save()
		_, err := SetReq(uu, o, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
	}
	for n := 0; n < 3; n++ {
		name := fmt.Sprintf("org-count-%d", n)
		ou, _ := organization.New(name, fmt.Sprintf("%s org", name))
		fakeacl.LoadFakeACL(ou)
		ou.Save()
		_, err := SetReq(u, ou, pivotal)
		if err != nil {
			t.Error(err.Error())
		}
	}
	orgCount, err := OrgsAssociationReqCount(u)
	if err != nil {
		t.Fatalf("OrgsAssociationReqCount failed: %s", err.Error())
	}
	if orgCount != 3 {
		t.Errorf("expected 3 org reqs, got %d", orgCount)
	}
	userCount, err := UsersAssociationReqCount(o)
	if err != nil {
		t.Fatalf("UsersAssociationReqCount failed: %s", err.Error())
	}
	if userCount != 4 {
		t.Errorf("expected 4 user reqs, got %d", userCount)
	}
}
