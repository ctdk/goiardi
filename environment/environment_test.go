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

package environment

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
	gob.Register(new(ChefEnvironment))
	gob.Register(map[string]interface{}{})
}

var envOrgCounter int

func setupEnvTestOrg(t *testing.T) *organization.Organization {
	gob.Register(new(organization.Organization))
	envOrgCounter++
	name := fmt.Sprintf("testenv-%d", envOrgCounter)
	org, err := orgloader.New(name, "Test Environment Org")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	return org
}

func TestEnvironmentNew(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, err := New(org, "prod")
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if env.Name != "prod" {
		t.Errorf("expected name 'prod', got %s", env.Name)
	}
	if env.ChefType != "environment" {
		t.Errorf("expected chef_type 'environment', got %s", env.ChefType)
	}
	if env.OrgName() != org.Name {
		t.Errorf("expected org %s, got %s", org.Name, env.OrgName())
	}
}

func TestEnvironmentNewInvalidName(t *testing.T) {
	org := setupEnvTestOrg(t)
	_, err := New(org, "bad name!")
	if err == nil {
		t.Fatal("expected error for invalid environment name")
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.Status())
	}
}

func TestEnvironmentNewDefaultConflict(t *testing.T) {
	org := setupEnvTestOrg(t)
	_, err := New(org, "_default")
	if err == nil {
		t.Fatal("expected error creating _default environment")
	}
}

func TestEnvironmentNewFromJSON(t *testing.T) {
	org := setupEnvTestOrg(t)
	jsonEnv := map[string]interface{}{
		"name":                "staging",
		"chef_type":           "environment",
		"json_class":          "Chef::Environment",
		"description":         "staging env",
		"default_attributes":  map[string]interface{}{"a": "b"},
		"override_attributes": map[string]interface{}{"c": "d"},
		"cookbook_versions":   map[string]interface{}{"apt": ">= 0.0.0"},
	}
	env, err := NewFromJSON(org, jsonEnv)
	if err != nil {
		t.Fatalf("NewFromJSON() failed: %s", err.Error())
	}
	if env.Name != "staging" {
		t.Errorf("expected name 'staging', got %s", env.Name)
	}
	if env.Description != "staging env" {
		t.Errorf("expected description 'staging env', got %s", env.Description)
	}
	if env.Default["a"] != "b" {
		t.Errorf("expected default attribute a=b, got %v", env.Default["a"])
	}
	if env.CookbookVersions["apt"] != ">= 0.0.0" {
		t.Errorf("expected apt constraint, got %s", env.CookbookVersions["apt"])
	}
}

func TestEnvironmentSaveAndGet(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := New(org, "saveenv")
	if err := env.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	env2, err := Get(org, "saveenv")
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if env2.Name != "saveenv" {
		t.Errorf("expected name 'saveenv', got %s", env2.Name)
	}
}

func TestEnvironmentGetDefault(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, err := Get(org, "_default")
	if err != nil {
		t.Fatalf("Get(_default) failed: %s", err.Error())
	}
	if env.Name != "_default" {
		t.Errorf("expected _default, got %s", env.Name)
	}
}

func TestEnvironmentGetNotFound(t *testing.T) {
	org := setupEnvTestOrg(t)
	_, err := Get(org, "missing")
	if err == nil {
		t.Fatal("expected error for missing environment")
	}
	if err.Status() != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, err.Status())
	}
}

func TestEnvironmentDoesExist(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := New(org, "exists")
	env.Save()
	found, err := DoesExist(org, "exists")
	if err != nil {
		t.Fatalf("DoesExist() failed: %s", err.Error())
	}
	if !found {
		t.Error("expected DoesExist true for saved environment")
	}
	found, _ = DoesExist(org, "nope")
	if found {
		t.Error("expected DoesExist false for missing environment")
	}
}

func TestEnvironmentGetMulti(t *testing.T) {
	org := setupEnvTestOrg(t)
	for _, name := range []string{"e1", "e2", "e3"} {
		e, _ := New(org, name)
		e.Save()
	}
	envs, err := GetMulti(org, []string{"e1", "e3"})
	if err != nil {
		t.Fatalf("GetMulti() failed: %s", err.Error())
	}
	if len(envs) != 2 {
		t.Errorf("expected 2 environments, got %d", len(envs))
	}
}

func TestEnvironmentGetList(t *testing.T) {
	org := setupEnvTestOrg(t)
	for _, name := range []string{"list-a", "list-b"} {
		e, _ := New(org, name)
		e.Save()
	}
	list := GetList(org)
	found := make(map[string]bool)
	for _, n := range list {
		found[n] = true
	}
	for _, name := range []string{"list-a", "list-b", "_default"} {
		if !found[name] {
			t.Errorf("expected %s in list, got %v", name, list)
		}
	}
}

func TestEnvironmentAllEnvironments(t *testing.T) {
	org := setupEnvTestOrg(t)
	for _, name := range []string{"all-a", "all-b"} {
		e, _ := New(org, name)
		e.Save()
	}
	all := AllEnvironments(org)
	if len(all) != 3 { // 2 custom + _default
		t.Errorf("expected 3 environments, got %d", len(all))
	}
}

func TestEnvironmentUpdateFromJSON(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := New(org, "updateenv")
	env.Save()
	env2, _ := Get(org, "updateenv")
	err := env2.UpdateFromJSON(map[string]interface{}{
		"name":                "updateenv",
		"chef_type":           "environment",
		"json_class":          "Chef::Environment",
		"description":         "updated",
		"default_attributes":  map[string]interface{}{},
		"override_attributes": map[string]interface{}{},
		"cookbook_versions":   map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("UpdateFromJSON() failed: %s", err.Error())
	}
	if err := env2.Save(); err != nil {
		t.Fatalf("Save() after update failed: %s", err.Error())
	}
	env3, _ := Get(org, "updateenv")
	if env3.Description != "updated" {
		t.Errorf("expected description 'updated', got %s", env3.Description)
	}
}

func TestEnvironmentDefaultCannotBeModified(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := Get(org, "_default")
	err := env.UpdateFromJSON(map[string]interface{}{
		"name":                "_default",
		"chef_type":           "environment",
		"json_class":          "Chef::Environment",
		"description":         "nope",
		"default_attributes":  map[string]interface{}{},
		"override_attributes": map[string]interface{}{},
		"cookbook_versions":   map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected error modifying _default environment")
	}
	if err.Status() != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, err.Status())
	}
}

func TestEnvironmentDelete(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := New(org, "deleteme")
	env.Save()
	if err := env.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, "deleteme")
	if err == nil {
		t.Fatal("expected error getting deleted environment")
	}
}

func TestEnvironmentDeleteDefault(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := Get(org, "_default")
	if err := env.Delete(); err == nil {
		t.Fatal("expected error deleting _default environment")
	}
}

func TestEnvironmentFlatten(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := New(org, "flatenv")
	env.Default = map[string]interface{}{"level1": map[string]interface{}{"level2": "val"}}
	flat := env.Flatten()
	if flat["default_attributes_level1_level2"] != "val" {
		t.Errorf("expected flattened default_attributes_level1_level2 'val', got %v", flat["default_attributes_level1_level2"])
	}
}

func TestEnvironmentDocIDAndIndex(t *testing.T) {
	org := setupEnvTestOrg(t)
	env, _ := New(org, "docidenv")
	if env.DocID() != "docidenv" {
		t.Errorf("expected DocID 'docidenv', got %s", env.DocID())
	}
	if env.Index() != "environment" {
		t.Errorf("expected Index 'environment', got %s", env.Index())
	}
}
