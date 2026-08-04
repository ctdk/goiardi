/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package user

import (
	"encoding/gob"
	"testing"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
)

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(User))
}

func resetUserDatastore() {
	ds := datastore.New()
	for _, name := range ds.GetList("user") {
		ds.Delete("user", name)
	}
}

func makeUserJSON(name string) map[string]interface{} {
	return map[string]interface{}{
		"username":     name,
		"name":         name,
		"display_name": name,
		"email":        name + "@example.com",
		"password":     "password123",
		"admin":        false,
		"first_name":   "Test",
		"last_name":    "User",
	}
}

func TestUserNew(t *testing.T) {
	resetUserDatastore()
	u, err := New("test-user")
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if u.Username != "test-user" {
		t.Errorf("expected username test-user, got %s", u.Username)
	}
}

func TestUserNewDuplicate(t *testing.T) {
	resetUserDatastore()
	u, _ := New("dup-user")
	u.Save()
	_, err := New("dup-user")
	if err == nil {
		t.Fatal("expected error for duplicate user")
	}
}

func TestUserNewInvalidName(t *testing.T) {
	resetUserDatastore()
	_, err := New("bad name")
	if err == nil {
		t.Fatal("expected error for invalid username")
	}
}

func TestUserSaveAndGet(t *testing.T) {
	resetUserDatastore()
	u, _ := New("save-user")
	if err := u.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	u2, err := Get("save-user")
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if u2.Username != "save-user" {
		t.Errorf("expected save-user, got %s", u2.Username)
	}
}

func TestUserGetNotFound(t *testing.T) {
	resetUserDatastore()
	_, err := Get("missing-user")
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestUserDoesExist(t *testing.T) {
	resetUserDatastore()
	u, _ := New("exists-user")
	u.Save()
	found, err := DoesExist(nil, "exists-user")
	if err != nil {
		t.Fatalf("DoesExist() failed: %s", err.Error())
	}
	if !found {
		t.Error("expected user to exist")
	}
}

func TestUserGetList(t *testing.T) {
	resetUserDatastore()
	for _, name := range []string{"list-a", "list-b"} {
		u, _ := New(name)
		u.Save()
	}
	list := GetList()
	if len(list) != 2 {
		t.Errorf("expected 2 users, got %d", len(list))
	}
}

func TestUserDelete(t *testing.T) {
	resetUserDatastore()
	u, _ := New("delete-user")
	u.Save()
	if err := u.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get("delete-user")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestUserNewFromJSON(t *testing.T) {
	resetUserDatastore()
	u, err := NewFromJSON(makeUserJSON("json-user"))
	if err != nil {
		t.Fatalf("NewFromJSON() failed: %s", err.Error())
	}
	if u.Username != "json-user" {
		t.Errorf("expected json-user, got %s", u.Username)
	}
}

func TestUserUpdateFromJSON(t *testing.T) {
	resetUserDatastore()
	u, _ := New("update-user")
	json := makeUserJSON("update-user")
	json["admin"] = true
	if err := u.UpdateFromJSON(json); err != nil {
		t.Fatalf("UpdateFromJSON() failed: %s", err.Error())
	}
	if !u.Admin {
		t.Error("expected admin true after update")
	}
}

func TestUserToJSON(t *testing.T) {
	resetUserDatastore()
	u, _ := New("jsonout-user")
	j := u.ToJSON()
	if j["username"] != "jsonout-user" {
		t.Errorf("expected username jsonout-user, got %v", j["username"])
	}
}

func TestUserIsUser(t *testing.T) {
	resetUserDatastore()
	u, _ := New("isuser-user")
	if !u.IsUser() {
		t.Error("user IsUser should be true")
	}
	if u.IsClient() {
		t.Error("user IsClient should be false")
	}
}

func TestUserURLType(t *testing.T) {
	u := &User{Username: "url-user"}
	if u.URLType() != "users" {
		t.Errorf("expected users, got %s", u.URLType())
	}
}

func TestUserSetAndCheckPasswd(t *testing.T) {
	resetUserDatastore()
	u, _ := New("passwd-user")
	if err := u.SetPasswd("secret123"); err != nil {
		t.Fatalf("SetPasswd() failed: %s", err.Error())
	}
	if err := u.CheckPasswd("secret123"); err != nil {
		t.Errorf("CheckPasswd with correct password failed: %s", err.Error())
	}
	if err := u.CheckPasswd("wrong"); err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestUserGetByEmail(t *testing.T) {
	resetUserDatastore()
	u, _ := New("email-user")
	u.Email = "email-user@example.com"
	u.Save()
	got, err := GetByEmail("email-user@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() failed: %s", err.Error())
	}
	if got.Username != "email-user" {
		t.Errorf("expected email-user, got %s", got.Username)
	}
}
