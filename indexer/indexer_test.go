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

package indexer

import (
	"testing"

	"github.com/ctdk/goiardi/config"
)

type fakeIndexable struct {
	name string
	idx  string
	org  string
}

func (f *fakeIndexable) DocID() string                { return f.name }
func (f *fakeIndexable) Index() string                  { return f.idx }
func (f *fakeIndexable) OrgName() string                { return f.org }
func (f *fakeIndexable) Flatten() map[string]interface{} { return map[string]interface{}{"name": f.name} }

type fakeOrg struct {
	name       string
	id         int64
	schemaName string
}

func (f *fakeOrg) GetName() string          { return f.name }
func (f *fakeOrg) GetId() int64             { return f.id }
func (f *fakeOrg) SearchSchemaName() string { return f.schemaName }

func resetIndexer() {
	Initialize(config.Config, DefaultDummyOrg)
}

func TestIndexerInitialize(t *testing.T) {
	resetIndexer()
	if GetIndex() == nil {
		t.Fatal("expected index to be initialized")
	}
}

func TestCreateOrgDex(t *testing.T) {
	resetIndexer()
	org := &fakeOrg{name: "dex-org", id: 42, schemaName: "goiardi_search_org_42"}
	if err := CreateOrgDex(org); err != nil {
		t.Fatalf("CreateOrgDex failed: %s", err.Error())
	}
	orgs := OrgList()
	found := false
	for _, o := range orgs {
		if o == org.name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %s in org list %v", org.name, orgs)
	}
}

func TestDeleteOrgDex(t *testing.T) {
	resetIndexer()
	org := &fakeOrg{name: "del-org", id: 43, schemaName: "goiardi_search_org_43"}
	if err := CreateOrgDex(org); err != nil {
		t.Fatalf("CreateOrgDex failed: %s", err.Error())
	}
	if err := DeleteOrgDex(org); err != nil {
		t.Fatalf("DeleteOrgDex failed: %s", err.Error())
	}
	for _, o := range OrgList() {
		if o == org.name {
			t.Errorf("expected %s to be deleted", org.name)
		}
	}
}

func TestCreateCollection(t *testing.T) {
	resetIndexer()
	org := &fakeOrg{name: "coll-org", id: 44, schemaName: "goiardi_search_org_44"}
	CreateOrgDex(org)
	CreateNewCollection(org, "test_bags")
	endpoints, err := Endpoints(org)
	if err != nil {
		t.Fatalf("Endpoints failed: %s", err.Error())
	}
	found := false
	for _, e := range endpoints {
		if e == "test_bags" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected test_bags in endpoints %v", endpoints)
	}
}

func TestDeleteCollection(t *testing.T) {
	resetIndexer()
	org := &fakeOrg{name: "del-coll-org", id: 45, schemaName: "goiardi_search_org_45"}
	CreateOrgDex(org)
	CreateNewCollection(org, "test_bags")
	if err := DeleteCollection(org, "test_bags"); err != nil {
		t.Fatalf("DeleteCollection failed: %s", err.Error())
	}
	endpoints, _ := Endpoints(org)
	for _, e := range endpoints {
		if e == "test_bags" {
			t.Errorf("expected test_bags to be deleted")
		}
	}
}

func TestDeleteDefaultCollection(t *testing.T) {
	resetIndexer()
	org := &fakeOrg{name: "def-coll-org", id: 46, schemaName: "goiardi_search_org_46"}
	CreateOrgDex(org)
	if err := DeleteCollection(org, "node"); err == nil {
		t.Error("expected error deleting default collection")
	}
}

func TestIndexObjAndSearch(t *testing.T) {
	resetIndexer()
	org := &fakeOrg{name: "search-org", id: 47, schemaName: "goiardi_search_org_47"}
	CreateOrgDex(org)
	obj := &fakeIndexable{name: "foo", idx: "node", org: org.name}
	// save synchronously
	objIndex.SaveItem(org, obj)

	results, err := GetIndex().Search(org, "node", "foo", true)
	if err != nil {
		t.Fatalf("Search failed: %s", err.Error())
	}
	if _, ok := results["foo"]; !ok {
		t.Errorf("expected foo in results %v", results)
	}

	all, err := GetIndex().Search(org, "node", "*:*", true)
	if err != nil {
		t.Fatalf("Search *:* failed: %s", err.Error())
	}
	if _, ok := all["foo"]; !ok {
		t.Errorf("expected foo in all results %v", all)
	}
}
