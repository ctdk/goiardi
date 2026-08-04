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

package node

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
	gob.Register(new(Node))
}

type noopNodePermChecker struct{}

func (n *noopNodePermChecker) CheckItemPerm(item aclhelper.Item, actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopNodePermChecker) CheckACLItemPerm(item aclhelper.Item, actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopNodePermChecker) CheckContainerPerm(actor aclhelper.Actor, container string, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopNodePermChecker) RootCheckPerm(actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopNodePermChecker) EditItemPerm(item aclhelper.Item, member aclhelper.Member, members []string, action string) util.Gerror {
	return nil
}
func (n *noopNodePermChecker) AddMembers(role aclhelper.Role, members []aclhelper.Member) error {
	return nil
}
func (n *noopNodePermChecker) RemoveMembers(role aclhelper.Role, members []aclhelper.Member) error {
	return nil
}
func (n *noopNodePermChecker) AddACLRole(role aclhelper.Role) error {
	return nil
}
func (n *noopNodePermChecker) RemoveACLRole(role aclhelper.Role) error {
	return nil
}
func (n *noopNodePermChecker) Enforcer() *casbin.SyncedEnforcer {
	return nil
}
func (n *noopNodePermChecker) GetItemACL(item aclhelper.Item) (*aclhelper.ACL, error) {
	return nil, nil
}
func (n *noopNodePermChecker) DeleteItemACL(item aclhelper.Item) (bool, error) {
	return true, nil
}
func (n *noopNodePermChecker) RenameItemACL(item aclhelper.Item, oldName string) error {
	return nil
}
func (n *noopNodePermChecker) EditFromJSON(item aclhelper.Item, action string, data interface{}) util.Gerror {
	return nil
}
func (n *noopNodePermChecker) CreatorOnly(item aclhelper.Item, actor aclhelper.Actor) util.Gerror {
	return nil
}
func (n *noopNodePermChecker) RemoveUser(member aclhelper.Member) error {
	return nil
}
func (n *noopNodePermChecker) RenameMember(member aclhelper.Member, newName string) error {
	return nil
}
func (n *noopNodePermChecker) DeletePolicy() error {
	return nil
}

var nodeOrgCounter int

func setupNodeTestOrg(t *testing.T) *organization.Organization {
	gob.Register(new(organization.Organization))
	nodeOrgCounter++
	name := fmt.Sprintf("testnode-%d", nodeOrgCounter)
	org, err := organization.New(name, "Test Node Org")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	org.SetPermCheck(&noopNodePermChecker{})
	return org
}

func makeNodeJSON(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":              name,
		"json_class":        "Chef::Node",
		"chef_type":         "node",
		"chef_environment":  "_default",
		"run_list":          []string{},
		"automatic":         map[string]interface{}{},
		"normal":            map[string]interface{}{},
		"default":           map[string]interface{}{},
		"override":          map[string]interface{}{},
	}
}

func TestNodeNew(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, err := New(org, "test-node")
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if n.Name != "test-node" {
		t.Errorf("expected name test-node, got %s", n.Name)
	}
}

func TestNodeNewInvalidName(t *testing.T) {
	org := setupNodeTestOrg(t)
	_, err := New(org, "bad name")
	if err == nil {
		t.Fatal("expected error for invalid node name")
	}
}

func TestNodeNewDuplicate(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, _ := New(org, "dup-node")
	n.Save()
	_, err := New(org, "dup-node")
	if err == nil {
		t.Fatal("expected error for duplicate node")
	}
}

func TestNodeSaveAndGet(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, _ := New(org, "save-node")
	if err := n.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	n2, err := Get(org, "save-node")
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if n2.Name != "save-node" {
		t.Errorf("expected save-node, got %s", n2.Name)
	}
}

func TestNodeGetNotFound(t *testing.T) {
	org := setupNodeTestOrg(t)
	_, err := Get(org, "missing-node")
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

func TestNodeDoesExist(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, _ := New(org, "exists-node")
	n.Save()
	found, err := DoesExist(org, "exists-node")
	if err != nil {
		t.Fatalf("DoesExist() failed: %s", err.Error())
	}
	if !found {
		t.Error("expected node to exist")
	}
}

func TestNodeGetMulti(t *testing.T) {
	org := setupNodeTestOrg(t)
	for _, name := range []string{"multi-a", "multi-b"} {
		n, _ := New(org, name)
		n.Save()
	}
	nodes, err := GetMulti(org, []string{"multi-a", "multi-b", "missing"})
	if err != nil {
		t.Fatalf("GetMulti() failed: %s", err.Error())
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestNodeGetList(t *testing.T) {
	org := setupNodeTestOrg(t)
	for _, name := range []string{"list-a", "list-b"} {
		n, _ := New(org, name)
		n.Save()
	}
	list := GetList(org)
	if len(list) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(list))
	}
}

func TestNodeDelete(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, _ := New(org, "delete-node")
	n.Save()
	if err := n.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, "delete-node")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestNodeNewFromJSON(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, err := NewFromJSON(org, makeNodeJSON("json-node"))
	if err != nil {
		t.Fatalf("NewFromJSON() failed: %s", err.Error())
	}
	if n.Name != "json-node" {
		t.Errorf("expected json-node, got %s", n.Name)
	}
}

func TestNodeUpdateFromJSON(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, _ := New(org, "update-node")
	json := makeNodeJSON("update-node")
	json["chef_environment"] = "production"
	if err := n.UpdateFromJSON(json); err != nil {
		t.Fatalf("UpdateFromJSON() failed: %s", err.Error())
	}
	if n.ChefEnvironment != "production" {
		t.Errorf("expected production, got %s", n.ChefEnvironment)
	}
}

func TestNodeFlatten(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, _ := New(org, "flatten-node")
	flat := n.Flatten()
	if _, ok := flat["name"]; !ok {
		t.Error("expected name in flattened node")
	}
}

func TestNodeAllNodes(t *testing.T) {
	org := setupNodeTestOrg(t)
	for _, name := range []string{"all-a", "all-b"} {
		n, _ := New(org, name)
		n.Save()
	}
	all := AllNodes(org)
	if len(all) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(all))
	}
}

func TestNodeGetFromEnv(t *testing.T) {
	org := setupNodeTestOrg(t)
	n, _ := New(org, "env-node")
	n.ChefEnvironment = "staging"
	n.Save()
	envNodes, err := GetFromEnv(org, "staging")
	if err != nil {
		t.Fatalf("GetFromEnv() failed: %s", err.Error())
	}
	if len(envNodes) != 1 {
		t.Errorf("expected 1 node in staging, got %d", len(envNodes))
	}
}

func TestNodeOrgCount(t *testing.T) {
	org := setupNodeTestOrg(t)
	for _, name := range []string{"count-a", "count-b"} {
		n, _ := New(org, name)
		n.Save()
	}
	if c := OrgCount(org); c != 2 {
		t.Errorf("expected 2 nodes, got %d", c)
	}
}

func TestNodeURLType(t *testing.T) {
	n := &Node{Name: "url-node", ChefType: "node"}
	if n.URLType() != "nodes" {
		t.Errorf("expected nodes, got %s", n.URLType())
	}
}
