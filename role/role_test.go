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

package role

import (
	"encoding/gob"
	"fmt"
	"net/http"
	"testing"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
)

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(Role))
	gob.Register(map[string]interface{}{})
	gob.Register(map[string][]string{})
}

var roleOrgCounter int

func setupRoleTestOrg(t *testing.T) *organization.Organization {
	gob.Register(new(organization.Organization))
	roleOrgCounter++
	name := fmt.Sprintf("testrole-%d", roleOrgCounter)
	org, err := orgloader.New(name, "Test Role Org")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	return org
}

func TestRoleNew(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, err := New(org, "web")
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if role.Name != "web" {
		t.Errorf("expected name 'web', got %s", role.Name)
	}
	if role.ChefType != "role" {
		t.Errorf("expected chef_type 'role', got %s", role.ChefType)
	}
	if role.OrgName() != org.Name {
		t.Errorf("expected org %s, got %s", org.Name, role.OrgName())
	}
}

func TestRoleNewInvalidName(t *testing.T) {
	org := setupRoleTestOrg(t)
	_, err := New(org, "bad name!")
	if err == nil {
		t.Fatal("expected error for invalid role name")
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.Status())
	}
}

func TestRoleNewDuplicate(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, _ := New(org, "dup")
	if err := role.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	_, err := New(org, "dup")
	if err == nil {
		t.Fatal("expected error for duplicate role")
	}
	if err.Status() != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, err.Status())
	}
}

func TestRoleNewFromJSON(t *testing.T) {
	org := setupRoleTestOrg(t)
	jsonRole := map[string]interface{}{
		"name":                "db",
		"chef_type":           "role",
		"json_class":          "Chef::Role",
		"run_list":            []string{"recipe[base]"},
		"env_run_lists":       map[string][]string{},
		"default_attributes":  map[string]interface{}{"a": "b"},
		"override_attributes": map[string]interface{}{"c": "d"},
		"description":         "db role",
	}
	role, err := NewFromJSON(org, jsonRole)
	if err != nil {
		t.Fatalf("NewFromJSON() failed: %s", err.Error())
	}
	if role.Name != "db" {
		t.Errorf("expected name 'db', got %s", role.Name)
	}
	if len(role.RunList) != 1 || role.RunList[0] != "recipe[base]" {
		t.Errorf("unexpected run_list: %v", role.RunList)
	}
	if role.Default["a"] != "b" {
		t.Errorf("expected default a=b, got %v", role.Default["a"])
	}
}

func TestRoleSaveAndGet(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, _ := New(org, "saveme")
	if err := role.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	role2, err := Get(org, "saveme")
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if role2.Name != "saveme" {
		t.Errorf("expected name 'saveme', got %s", role2.Name)
	}
}

func TestRoleGetNotFound(t *testing.T) {
	org := setupRoleTestOrg(t)
	_, err := Get(org, "missing")
	if err == nil {
		t.Fatal("expected error for missing role")
	}
	if err.Status() != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, err.Status())
	}
}

func TestRoleDoesExist(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, _ := New(org, "exists")
	role.Save()
	found, err := DoesExist(org, "exists")
	if err != nil {
		t.Fatalf("DoesExist() failed: %s", err.Error())
	}
	if !found {
		t.Error("expected DoesExist true for saved role")
	}
	found, _ = DoesExist(org, "nope")
	if found {
		t.Error("expected DoesExist false for missing role")
	}
}

func TestRoleGetMulti(t *testing.T) {
	org := setupRoleTestOrg(t)
	for _, name := range []string{"r1", "r2", "r3"} {
		r, _ := New(org, name)
		r.Save()
	}
	roles, err := GetMulti(org, []string{"r1", "r3"})
	if err != nil {
		t.Fatalf("GetMulti() failed: %s", err.Error())
	}
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestRoleGetList(t *testing.T) {
	org := setupRoleTestOrg(t)
	for _, name := range []string{"list-a", "list-b"} {
		r, _ := New(org, name)
		r.Save()
	}
	list := GetList(org)
	if len(list) != 2 {
		t.Errorf("expected 2 roles, got %d", len(list))
	}
}

func TestRoleAllRoles(t *testing.T) {
	org := setupRoleTestOrg(t)
	for _, name := range []string{"all-a", "all-b"} {
		r, _ := New(org, name)
		r.Save()
	}
	all := AllRoles(org)
	if len(all) != 2 {
		t.Errorf("expected 2 roles, got %d", len(all))
	}
}

func TestRoleUpdateFromJSON(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, _ := New(org, "updateme")
	role.Save()
	role2, _ := Get(org, "updateme")
	err := role2.UpdateFromJSON(map[string]interface{}{
		"name":                "updateme",
		"chef_type":           "role",
		"json_class":          "Chef::Role",
		"run_list":            []string{"recipe[new]"},
		"env_run_lists":       map[string][]string{},
		"default_attributes":  map[string]interface{}{},
		"override_attributes": map[string]interface{}{},
		"description":         "updated",
	})
	if err != nil {
		t.Fatalf("UpdateFromJSON() failed: %s", err.Error())
	}
	if err := role2.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	role3, _ := Get(org, "updateme")
	if role3.Description != "updated" {
		t.Errorf("expected description 'updated', got %s", role3.Description)
	}
	if len(role3.RunList) != 1 || role3.RunList[0] != "recipe[new]" {
		t.Errorf("expected run_list [recipe[new]], got %v", role3.RunList)
	}
}

func TestRoleDelete(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, _ := New(org, "deleteme")
	role.Save()
	if err := role.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, "deleteme")
	if err == nil {
		t.Fatal("expected error getting deleted role")
	}
}

func TestRoleFlatten(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, _ := New(org, "flat")
	role.Default = map[string]interface{}{"level1": map[string]interface{}{"level2": "val"}}
	flat := role.Flatten()
	if flat["default_attributes_level1_level2"] != "val" {
		t.Errorf("expected flattened default_attributes_level1_level2 'val', got %v", flat["default_attributes_level1_level2"])
	}
}

func TestRoleDocIDAndIndex(t *testing.T) {
	org := setupRoleTestOrg(t)
	role, _ := New(org, "docid")
	if role.DocID() != "docid" {
		t.Errorf("expected DocID 'docid', got %s", role.DocID())
	}
	if role.Index() != "role" {
		t.Errorf("expected Index 'role', got %s", role.Index())
	}
	if role.URLType() != "roles" {
		t.Errorf("expected URLType 'roles', got %s", role.URLType())
	}
}
