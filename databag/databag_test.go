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

package databag

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
	gob.Register(new(DataBag))
}

var orgCounter int

func setupTestOrg(t *testing.T) *organization.Organization {
	gob.Register(new(organization.Organization))
	orgCounter++
	name := fmt.Sprintf("testdatabag-%d", orgCounter)
	org, err := orgloader.New(name, "Test Data Bag Org")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	return org
}

func TestDataBagNew(t *testing.T) {
	org := setupTestOrg(t)
	db, err := New(org, "testbag")
	if err != nil {
		t.Fatalf("New() returned error: %s", err.Error())
	}
	if db.Name != "testbag" {
		t.Errorf("expected data bag name 'testbag', got %s", db.Name)
	}
	if db.OrgName() != org.Name {
		t.Errorf("expected org name %s, got %s", org.Name, db.OrgName())
	}
	if db.DataBagItems == nil {
		t.Errorf("expected DataBagItems map to be initialized")
	}
}

func TestDataBagNewDuplicate(t *testing.T) {
	org := setupTestOrg(t)
	db, err := New(org, "dupbag")
	if err != nil {
		t.Fatalf("first New() failed: %s", err.Error())
	}
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	_, err = New(org, "dupbag")
	if err == nil {
		t.Fatal("expected error creating duplicate data bag, got nil")
	}
	if err.Status() != http.StatusConflict {
		t.Errorf("expected status %d for duplicate, got %d", http.StatusConflict, err.Status())
	}
}

func TestDataBagNewInvalidName(t *testing.T) {
	org := setupTestOrg(t)
	_, err := New(org, "bad name!")
	if err == nil {
		t.Fatal("expected error for invalid data bag name, got nil")
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid name, got %d", http.StatusBadRequest, err.Status())
	}
}

func TestDataBagSaveAndGet(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "savebag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}

	db2, err := Get(org, "savebag")
	if err != nil {
		t.Fatalf("Get() failed after Save(): %s", err.Error())
	}
	if db2.Name != "savebag" {
		t.Errorf("expected name 'savebag', got %s", db2.Name)
	}
}

func TestDataBagDelete(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "delbag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	if err := db.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, "delbag")
	if err == nil {
		t.Fatal("expected error getting deleted data bag, got nil")
	}
	if err.Status() != http.StatusNotFound {
		t.Errorf("expected status %d for deleted bag, got %d", http.StatusNotFound, err.Status())
	}
}

func TestDataBagDoesExist(t *testing.T) {
	org := setupTestOrg(t)
	found, err := DoesExist(org, "missing")
	if err != nil {
		t.Fatalf("DoesExist() returned error for missing bag: %s", err.Error())
	}
	if found {
		t.Error("expected DoesExist to be false for missing bag")
	}

	db, _ := New(org, "existing")
	if serr := db.Save(); serr != nil {
		t.Fatalf("Save() failed: %s", serr.Error())
	}
	found, err = DoesExist(org, "existing")
	if err != nil {
		t.Fatalf("DoesExist() returned error for existing bag: %s", err.Error())
	}
	if !found {
		t.Error("expected DoesExist to be true for existing bag")
	}
}

func TestDataBagGetList(t *testing.T) {
	org := setupTestOrg(t)
	for _, name := range []string{"bag-a", "bag-b", "bag-c"} {
		db, _ := New(org, name)
		if err := db.Save(); err != nil {
			t.Fatalf("Save() failed for %s: %s", name, err.Error())
		}
	}

	list := GetList(org)
	if len(list) != 3 {
		t.Fatalf("expected 3 data bags, got %d", len(list))
	}
	for _, name := range []string{"bag-a", "bag-b", "bag-c"} {
		found := false
		for _, n := range list {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s in list, got %v", name, list)
		}
	}
}

func TestDataBagAllDataBags(t *testing.T) {
	org := setupTestOrg(t)
	for _, name := range []string{"all-1", "all-2"} {
		db, _ := New(org, name)
		if err := db.Save(); err != nil {
			t.Fatalf("Save() failed for %s: %s", name, err.Error())
		}
	}

	all := AllDataBags(org)
	if len(all) != 2 {
		t.Fatalf("expected 2 data bags, got %d", len(all))
	}
	for _, db := range all {
		if db.Name != "all-1" && db.Name != "all-2" {
			t.Errorf("unexpected data bag %s in AllDataBags", db.Name)
		}
	}
}

func TestDataBagGetNotFound(t *testing.T) {
	org := setupTestOrg(t)
	_, err := Get(org, "never-existed")
	if err == nil {
		t.Fatal("expected error for missing data bag, got nil")
	}
	if err.Status() != http.StatusNotFound {
		t.Errorf("expected status %d for missing bag, got %d", http.StatusNotFound, err.Status())
	}
}

func TestDataBagGetNameURLType(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "infobag")
	if db.GetName() != "infobag" {
		t.Errorf("expected GetName() 'infobag', got %s", db.GetName())
	}
	if db.URLType() != "data" {
		t.Errorf("expected URLType() 'data', got %s", db.URLType())
	}
	if db.ContainerType() != "data" {
		t.Errorf("expected ContainerType() 'data', got %s", db.ContainerType())
	}
	if db.ContainerKind() != "containers" {
		t.Errorf("expected ContainerKind() 'containers', got %s", db.ContainerKind())
	}
}
