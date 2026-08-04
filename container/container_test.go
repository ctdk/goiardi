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

package container

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
	gob.Register(new(Container))
}

var containerOrgCounter int

func setupContainerTestOrg(t *testing.T) *organization.Organization {
	gob.Register(new(organization.Organization))
	containerOrgCounter++
	name := fmt.Sprintf("testcontainer-%d", containerOrgCounter)
	org, err := orgloader.New(name, "Test Container Org")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	return org
}

func TestContainerNew(t *testing.T) {
	org := setupContainerTestOrg(t)
	c, err := New(org, "custom")
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if c.Name != "custom" {
		t.Errorf("expected name 'custom', got %s", c.Name)
	}
	if c.OrgName() != org.Name {
		t.Errorf("expected org %s, got %s", org.Name, c.OrgName())
	}
}

func TestContainerNewInvalidName(t *testing.T) {
	org := setupContainerTestOrg(t)
	_, err := New(org, "bad name!")
	if err == nil {
		t.Fatal("expected error for invalid container name")
	}
}

func TestContainerNewDuplicate(t *testing.T) {
	org := setupContainerTestOrg(t)
	c, _ := New(org, "dup")
	if err := c.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	_, err := New(org, "dup")
	if err == nil {
		t.Fatal("expected error for duplicate container")
	}
	if err.Status() != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, err.Status())
	}
}

func TestContainerSaveAndGet(t *testing.T) {
	org := setupContainerTestOrg(t)
	c, _ := New(org, "saveme")
	if err := c.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	c2, err := Get(org, "saveme")
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if c2.Name != "saveme" {
		t.Errorf("expected name 'saveme', got %s", c2.Name)
	}
}

func TestContainerGetNotFound(t *testing.T) {
	org := setupContainerTestOrg(t)
	_, err := Get(org, "missing")
	if err == nil {
		t.Fatal("expected error for missing container")
	}
	if err.Status() != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, err.Status())
	}
}

func TestContainerDelete(t *testing.T) {
	org := setupContainerTestOrg(t)
	c, _ := New(org, "deleteme")
	c.Save()
	if err := c.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, "deleteme")
	if err == nil {
		t.Fatal("expected error getting deleted container")
	}
}

func TestContainerGetList(t *testing.T) {
	org := setupContainerTestOrg(t)
	for _, name := range []string{"a", "b", "c"} {
		c, _ := New(org, name)
		c.Save()
	}
	list := GetList(org)
	if len(list) != 3 {
		t.Errorf("expected 3 containers, got %d", len(list))
	}
}

func TestContainerMakeDefaultContainers(t *testing.T) {
	org := setupContainerTestOrg(t)
	if err := MakeDefaultContainers(org); err != nil {
		t.Fatalf("MakeDefaultContainers() failed: %s", err.Error())
	}
	list := GetList(org)
	if len(list) != len(DefaultContainers) {
		t.Errorf("expected %d default containers, got %d", len(DefaultContainers), len(list))
	}
	for _, name := range DefaultContainers {
		found := false
		for _, n := range list {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected default container %s in list", name)
		}
	}
}

func TestContainerMetadata(t *testing.T) {
	org := setupContainerTestOrg(t)
	c, _ := New(org, "meta")
	if c.GetName() != "meta" {
		t.Errorf("expected GetName 'meta', got %s", c.GetName())
	}
	if c.URLType() != "containers" {
		t.Errorf("expected URLType 'containers', got %s", c.URLType())
	}
	if c.ContainerType() != "containers" {
		t.Errorf("expected ContainerType 'containers', got %s", c.ContainerType())
	}
	if c.ContainerKind() != "containers" {
		t.Errorf("expected ContainerKind 'containers', got %s", c.ContainerKind())
	}
}
