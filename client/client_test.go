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

package client

import (
	"encoding/gob"
	"fmt"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
)

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(Client))
}

var clientOrgCounter int

type noopPermChecker struct{}

func (n *noopPermChecker) CheckItemPerm(item aclhelper.Item, actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopPermChecker) CheckACLItemPerm(item aclhelper.Item, actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopPermChecker) CheckContainerPerm(actor aclhelper.Actor, container string, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopPermChecker) RootCheckPerm(actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopPermChecker) EditItemPerm(item aclhelper.Item, member aclhelper.Member, members []string, action string) util.Gerror {
	return nil
}
func (n *noopPermChecker) AddMembers(role aclhelper.Role, members []aclhelper.Member) error {
	return nil
}
func (n *noopPermChecker) RemoveMembers(role aclhelper.Role, members []aclhelper.Member) error {
	return nil
}
func (n *noopPermChecker) AddACLRole(role aclhelper.Role) error {
	return nil
}
func (n *noopPermChecker) RemoveACLRole(role aclhelper.Role) error {
	return nil
}
func (n *noopPermChecker) Enforcer() *casbin.SyncedEnforcer {
	return nil
}
func (n *noopPermChecker) GetItemACL(item aclhelper.Item) (*aclhelper.ACL, error) {
	return nil, nil
}
func (n *noopPermChecker) DeleteItemACL(item aclhelper.Item) (bool, error) {
	return true, nil
}
func (n *noopPermChecker) RenameItemACL(item aclhelper.Item, oldName string) error {
	return nil
}
func (n *noopPermChecker) EditFromJSON(item aclhelper.Item, action string, data interface{}) util.Gerror {
	return nil
}
func (n *noopPermChecker) CreatorOnly(item aclhelper.Item, actor aclhelper.Actor) util.Gerror {
	return nil
}
func (n *noopPermChecker) RemoveUser(member aclhelper.Member) error {
	return nil
}
func (n *noopPermChecker) RenameMember(member aclhelper.Member, newName string) error {
	return nil
}
func (n *noopPermChecker) DeletePolicy() error {
	return nil
}

func setupClientTestOrg(t *testing.T) *organization.Organization {
	gob.Register(new(organization.Organization))
	clientOrgCounter++
	name := fmt.Sprintf("testclient-%d", clientOrgCounter)
	org, err := organization.New(name, "Test Client Org")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	org.SetPermCheck(&noopPermChecker{})
	return org
}

func TestClientNew(t *testing.T) {
	org := setupClientTestOrg(t)
	c, err := New(org, "test-client")
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if c.Name != "test-client" {
		t.Errorf("expected name test-client, got %s", c.Name)
	}
}

func TestClientNewDuplicate(t *testing.T) {
	org := setupClientTestOrg(t)
	c, _ := New(org, "dup-client")
	c.Save()
	_, err := New(org, "dup-client")
	if err == nil {
		t.Fatal("expected error for duplicate client")
	}
}

func TestClientNewInvalidName(t *testing.T) {
	org := setupClientTestOrg(t)
	_, err := New(org, "bad name")
	if err == nil {
		t.Fatal("expected error for invalid client name")
	}
}

func TestClientSaveAndGet(t *testing.T) {
	org := setupClientTestOrg(t)
	c, _ := New(org, "save-client")
	if err := c.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	c2, err := Get(org, "save-client")
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if c2.Name != "save-client" {
		t.Errorf("expected save-client, got %s", c2.Name)
	}
}

func TestClientGetNotFound(t *testing.T) {
	org := setupClientTestOrg(t)
	_, err := Get(org, "missing-client")
	if err == nil {
		t.Fatal("expected error for missing client")
	}
}

func TestClientDoesExist(t *testing.T) {
	org := setupClientTestOrg(t)
	c, _ := New(org, "exists-client")
	c.Save()
	found, err := DoesExist(org, "exists-client")
	if err != nil {
		t.Fatalf("DoesExist() failed: %s", err.Error())
	}
	if !found {
		t.Error("expected client to exist")
	}
}

func TestClientGetMulti(t *testing.T) {
	org := setupClientTestOrg(t)
	for _, name := range []string{"multi-a", "multi-b"} {
		c, _ := New(org, name)
		c.Save()
	}
	clients, err := GetMulti(org, []string{"multi-a", "multi-b", "missing"})
	if err != nil {
		t.Fatalf("GetMulti() failed: %s", err.Error())
	}
	if len(clients) != 2 {
		t.Errorf("expected 2 clients, got %d", len(clients))
	}
}

func TestClientGetList(t *testing.T) {
	org := setupClientTestOrg(t)
	for _, name := range []string{"list-a", "list-b"} {
		c, _ := New(org, name)
		c.Save()
	}
	list := GetList(org)
	if len(list) != 2 {
		t.Errorf("expected 2 clients, got %d", len(list))
	}
}

func TestClientDelete(t *testing.T) {
	org := setupClientTestOrg(t)
	c, _ := New(org, "delete-client")
	c.Save()
	if err := c.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, "delete-client")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestClientNewFromJSON(t *testing.T) {
	org := setupClientTestOrg(t)
	json := map[string]interface{}{
		"name":       "json-client",
		"json_class": "Chef::ApiClient",
		"chef_type":  "client",
		"admin":      false,
		"validator":  false,
	}
	c, err := NewFromJSON(org, json)
	if err != nil {
		t.Fatalf("NewFromJSON() failed: %s", err.Error())
	}
	if c.Name != "json-client" {
		t.Errorf("expected json-client, got %s", c.Name)
	}
}

func TestClientUpdateFromJSON(t *testing.T) {
	org := setupClientTestOrg(t)
	c, _ := New(org, "update-client")
	json := map[string]interface{}{
		"name":       "update-client",
		"json_class": "Chef::ApiClient",
		"chef_type":  "client",
		"admin":      true,
		"validator":  false,
	}
	if err := c.UpdateFromJSON(json); err != nil {
		t.Fatalf("UpdateFromJSON() failed: %s", err.Error())
	}
	if !c.Admin {
		t.Error("expected admin true after update")
	}
}

func TestClientToJSON(t *testing.T) {
	org := setupClientTestOrg(t)
	c, _ := New(org, "jsonout-client")
	j := c.ToJSON()
	if j["name"] != "jsonout-client" {
		t.Errorf("expected name jsonout-client, got %v", j["name"])
	}
}

func TestClientIsUser(t *testing.T) {
	org := setupClientTestOrg(t)
	c, _ := New(org, "isuser-client")
	if c.IsUser() {
		t.Error("client IsUser should be false")
	}
	if !c.IsClient() {
		t.Error("client IsClient should be true")
	}
}

func TestClientURLType(t *testing.T) {
	c := &Client{Name: "url-client", ChefType: "client"}
	if c.URLType() != "clients" {
		t.Errorf("expected clients, got %s", c.URLType())
	}
}
