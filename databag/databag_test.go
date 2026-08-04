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
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
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
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
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

func makeJSONReader(v interface{}) io.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func makeDBItem(name string, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{"key": "value"}
	}
	data["id"] = name
	return data
}

func TestDataBagItemNew(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "itembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}

	dbi, err := db.NewDBItem(makeDBItem("item1", map[string]interface{}{"foo": "bar"}))
	if err != nil {
		t.Fatalf("NewDBItem() failed: %s", err.Error())
	}
	if dbi == nil {
		t.Fatal("NewDBItem() returned nil")
	}
	if dbi.DocID() != "item1" {
		t.Errorf("expected DocID 'item1', got %s", dbi.DocID())
	}
	if dbi.DataBagName != "itembag" {
		t.Errorf("expected DataBagName 'itembag', got %s", dbi.DataBagName)
	}
	if dbi.GetName() != "item1" {
		t.Errorf("expected GetName 'item1', got %s", dbi.GetName())
	}
	if dbi.Index() != "itembag" {
		t.Errorf("expected Index 'itembag', got %s", dbi.Index())
	}
	if dbi.RawData["foo"] != "bar" {
		t.Errorf("expected raw_data.foo 'bar', got %v", dbi.RawData["foo"])
	}
	if db.NumDBItems() != 1 {
		t.Errorf("expected 1 item, got %d", db.NumDBItems())
	}
}

func TestDataBagItemNewDuplicate(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "dupitembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	if _, err := db.NewDBItem(makeDBItem("dup", nil)); err != nil {
		t.Fatalf("first NewDBItem() failed: %s", err.Error())
	}
	_, err := db.NewDBItem(makeDBItem("dup", nil))
	if err == nil {
		t.Fatal("expected error creating duplicate data bag item")
	}
	if err.Status() != http.StatusConflict {
		t.Errorf("expected status %d for duplicate item, got %d", http.StatusConflict, err.Status())
	}
}

func TestDataBagItemNewMissingID(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "missingidbag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	_, err := db.NewDBItem(map[string]interface{}{"foo": "bar"})
	if err == nil {
		t.Fatal("expected error for missing item id")
	}
}

func TestDataBagItemNewInvalidName(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "invaliditembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	_, err := db.NewDBItem(makeDBItem("bad name!", nil))
	if err == nil {
		t.Fatal("expected error for invalid item id")
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("expected status %d for invalid item name, got %d", http.StatusBadRequest, err.Status())
	}
}

func TestDataBagItemGet(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "getitembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	if _, err := db.NewDBItem(makeDBItem("getme", map[string]interface{}{"a": "one"})); err != nil {
		t.Fatalf("NewDBItem() failed: %s", err.Error())
	}

	db2, _ := Get(org, "getitembag")
	dbi, err := db2.GetDBItem("getme")
	if err != nil {
		t.Fatalf("GetDBItem() failed: %s", err.Error())
	}
	if dbi.DocID() != "getme" {
		t.Errorf("expected DocID 'getme', got %s", dbi.DocID())
	}
	if dbi.RawData["a"] != "one" {
		t.Errorf("expected raw_data.a 'one', got %v", dbi.RawData["a"])
	}
}

func TestDataBagItemUpdate(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "updateitembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	dbi, _ := db.NewDBItem(makeDBItem("updateme", map[string]interface{}{"a": "original"}))
	if dbi.RawData["a"] != "original" {
		t.Errorf("expected original value 'original', got %v", dbi.RawData["a"])
	}

	db2, _ := Get(org, "updateitembag")
	dbi2, err := db2.UpdateDBItem("updateme", map[string]interface{}{"id": "updateme", "b": "updated"})
	if err != nil {
		t.Fatalf("UpdateDBItem() failed: %s", err.Error())
	}
	if dbi2.RawData["b"] != "updated" {
		t.Errorf("expected updated value 'updated', got %v", dbi2.RawData["b"])
	}
	if _, ok := dbi2.RawData["a"]; ok {
		t.Errorf("expected old key 'a' to be gone after update")
	}
}

func TestDataBagItemDelete(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "delitembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	db.NewDBItem(makeDBItem("delme", nil))

	db2, _ := Get(org, "delitembag")
	if err := db2.DeleteDBItem("delme"); err != nil {
		t.Fatalf("DeleteDBItem() failed: %s", err.Error())
	}
	if _, err := db2.GetDBItem("delme"); err == nil {
		t.Fatal("expected error getting deleted data bag item")
	}
	if n := db2.NumDBItems(); n != 0 {
		t.Errorf("expected 0 items after delete, got %d", n)
	}
}

func TestDataBagItemDoesExist(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "existitembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	found, err := db.DoesItemExist(org, "missing")
	if err != nil {
		t.Fatalf("DoesItemExist() returned error: %s", err.Error())
	}
	if found {
		t.Error("expected DoesItemExist false for missing item")
	}
	db.NewDBItem(makeDBItem("present", nil))
	found, err = db.DoesItemExist(org, "present")
	if err != nil {
		t.Fatalf("DoesItemExist() returned error: %s", err.Error())
	}
	if !found {
		t.Error("expected DoesItemExist true for existing item")
	}
}

func TestDataBagItemGetMultiAllList(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "multiitembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	for _, name := range []string{"one", "two", "three"} {
		if _, err := db.NewDBItem(makeDBItem(name, nil)); err != nil {
			t.Fatalf("NewDBItem(%s) failed: %s", name, err.Error())
		}
	}

	db2, _ := Get(org, "multiitembag")
	multi, err := db2.GetMultiDBItems([]string{"one", "three"})
	if err != nil {
		t.Fatalf("GetMultiDBItems() failed: %s", err.Error())
	}
	if len(multi) != 2 {
		t.Errorf("expected 2 items from multi-get, got %d", len(multi))
	}

	all, aerr := db2.AllDBItems()
	if aerr != nil {
		t.Fatalf("AllDBItems() failed: %s", aerr.Error())
	}
	if len(all) != 3 {
		t.Errorf("expected 3 items from AllDBItems, got %d", len(all))
	}

	list := db2.ListDBItems()
	if len(list) != 3 {
		t.Errorf("expected 3 items from ListDBItems, got %d", len(list))
	}
	for _, name := range []string{"one", "two", "three"} {
		found := false
		for _, n := range list {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s in ListDBItems, got %v", name, list)
		}
	}
}

func TestDataBagItemFlatten(t *testing.T) {
	org := setupTestOrg(t)
	db, _ := New(org, "flatitembag")
	if err := db.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	dbi, _ := db.NewDBItem(map[string]interface{}{
		"id": "flat",
		"nested": map[string]interface{}{
			"key": "val",
		},
	})

	flat := dbi.Flatten()
	if flat["nested_key"] != "val" {
		t.Errorf("expected flattened nested_key 'val', got %v", flat["nested_key"])
	}
}

func TestDataBagItemRawDataBagJSON(t *testing.T) {
	raw := map[string]interface{}{
		"id":       "jsonitem",
		"raw_data": map[string]interface{}{"a": 1},
	}
	data := RawDataBagJSON(io.NopCloser(makeJSONReader(raw)))
	if data["a"] != json.Number("1") {
		t.Errorf("expected raw_data.a 1, got %v (type %T)", data["a"], data["a"])
	}

	raw2 := map[string]interface{}{"b": 2}
	data2 := RawDataBagJSON(io.NopCloser(makeJSONReader(raw2)))
	if data2["b"] != json.Number("2") {
		t.Errorf("expected b 2, got %v (type %T)", data2["b"], data2["b"])
	}
}
