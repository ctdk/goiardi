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

package organization

import (
	"encoding/gob"
	"fmt"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
)

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(Organization))
}

type noopOrgPermChecker struct{}

func (n *noopOrgPermChecker) CheckItemPerm(item aclhelper.Item, actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopOrgPermChecker) CheckACLItemPerm(item aclhelper.Item, actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopOrgPermChecker) CheckContainerPerm(actor aclhelper.Actor, container string, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopOrgPermChecker) RootCheckPerm(actor aclhelper.Actor, action string) (bool, util.Gerror) {
	return true, nil
}
func (n *noopOrgPermChecker) EditItemPerm(item aclhelper.Item, member aclhelper.Member, members []string, action string) util.Gerror {
	return nil
}
func (n *noopOrgPermChecker) AddMembers(role aclhelper.Role, members []aclhelper.Member) error {
	return nil
}
func (n *noopOrgPermChecker) RemoveMembers(role aclhelper.Role, members []aclhelper.Member) error {
	return nil
}
func (n *noopOrgPermChecker) AddACLRole(role aclhelper.Role) error {
	return nil
}
func (n *noopOrgPermChecker) RemoveACLRole(role aclhelper.Role) error {
	return nil
}
func (n *noopOrgPermChecker) Enforcer() *casbin.SyncedEnforcer {
	return nil
}
func (n *noopOrgPermChecker) GetItemACL(item aclhelper.Item) (*aclhelper.ACL, error) {
	return nil, nil
}
func (n *noopOrgPermChecker) DeleteItemACL(item aclhelper.Item) (bool, error) {
	return true, nil
}
func (n *noopOrgPermChecker) RenameItemACL(item aclhelper.Item, oldName string) error {
	return nil
}
func (n *noopOrgPermChecker) EditFromJSON(item aclhelper.Item, action string, data interface{}) util.Gerror {
	return nil
}
func (n *noopOrgPermChecker) CreatorOnly(item aclhelper.Item, actor aclhelper.Actor) util.Gerror {
	return nil
}
func (n *noopOrgPermChecker) RemoveUser(member aclhelper.Member) error {
	return nil
}
func (n *noopOrgPermChecker) RenameMember(member aclhelper.Member, newName string) error {
	return nil
}
func (n *noopOrgPermChecker) DeletePolicy() error {
	return nil
}

var orgCounter int

func setupTestOrg(t *testing.T) *Organization {
	orgCounter++
	name := fmt.Sprintf("testorg-%d", orgCounter)
	org, err := New(name, "Test Organization")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	org.SetPermCheck(&noopOrgPermChecker{})
	return org
}

func TestOrganizationNew(t *testing.T) {
	org := setupTestOrg(t)
	if org.Name == "" {
		t.Fatal("expected non-empty org name")
	}
}

func TestOrganizationNewDuplicate(t *testing.T) {
	org := setupTestOrg(t)
	_, err := New(org.Name, "Duplicate")
	if err == nil {
		t.Fatal("expected error for duplicate org")
	}
}

func TestOrganizationSaveAndGet(t *testing.T) {
	org := setupTestOrg(t)
	if err := org.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	org2, err := Get(org.Name)
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if org2.Name != org.Name {
		t.Errorf("expected %s, got %s", org.Name, org2.Name)
	}
}

func TestOrganizationGetNotFound(t *testing.T) {
	_, err := Get("missing-org-12345")
	if err == nil {
		t.Fatal("expected error for missing org")
	}
}

func TestOrganizationGetList(t *testing.T) {
	setupTestOrg(t)
	setupTestOrg(t)
	list := GetList()
	if len(list) < 2 {
		t.Errorf("expected at least 2 orgs, got %d", len(list))
	}
}

func TestOrganizationAllOrganizations(t *testing.T) {
	setupTestOrg(t)
	setupTestOrg(t)
	orgs, err := AllOrganizations()
	if err != nil {
		t.Fatalf("AllOrganizations() failed: %s", err.Error())
	}
	if len(orgs) < 2 {
		t.Errorf("expected at least 2 orgs, got %d", len(orgs))
	}
}

func TestOrganizationToJSON(t *testing.T) {
	org := setupTestOrg(t)
	j := org.ToJSON()
	if j["name"] != org.Name {
		t.Errorf("expected name %s, got %v", org.Name, j["name"])
	}
}

func TestOrganizationDataKey(t *testing.T) {
	org := setupTestOrg(t)
	key := org.DataKey("client")
	expected := fmt.Sprintf("client-%s", org.Name)
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestOrganizationOrgURLBase(t *testing.T) {
	org := setupTestOrg(t)
	base := org.OrgURLBase()
	expected := fmt.Sprintf("/organizations/%s", org.Name)
	if base != expected {
		t.Errorf("expected %s, got %s", expected, base)
	}
}

func TestOrganizationContainerType(t *testing.T) {
	org := setupTestOrg(t)
	if org.ContainerType() != "$$root$$" {
		t.Errorf("expected $$root$$, got %s", org.ContainerType())
	}
}

func TestOrganizationDelete(t *testing.T) {
	org := setupTestOrg(t)
	if err := org.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org.Name)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
