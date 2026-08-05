/*
 * Copyright (c) 2013-2020, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package policy

import (
	"testing"
)

func revJSONFor(revID string) map[string]interface{} {
	return map[string]interface{}{
		"revision_id":           revID,
		"run_list":              []string{},
		"cookbook_locks":        map[string]interface{}{},
		"default_attributes":    map[string]interface{}{},
		"override_attributes":   map[string]interface{}{},
		"solution_dependencies": map[string]interface{}{},
	}
}

func TestPolicyDoesExistAndGetOrCreate(t *testing.T) {
	polName := "policy-exists-test"
	p, err := New(org, polName)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := p.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	found, err := DoesPolicyExist(org, polName)
	if err != nil {
		t.Fatalf("DoesPolicyExist failed: %v", err)
	}
	if !found {
		t.Errorf("expected policy %s to exist", polName)
	}

	// GetOrCreatePolicy should return existing
	p2, err := GetOrCreatePolicy(org, polName)
	if err != nil {
		t.Fatalf("GetOrCreatePolicy existing failed: %v", err)
	}
	if p2.Name != polName {
		t.Errorf("expected %s, got %s", polName, p2.Name)
	}

	// GetOrCreatePolicy should create a new one
	newName := "policy-get-or-create-new"
	p3, err := GetOrCreatePolicy(org, newName)
	if err != nil {
		t.Fatalf("GetOrCreatePolicy new failed: %v", err)
	}
	if p3.Name != newName {
		t.Errorf("expected %s, got %s", newName, p3.Name)
	}
}

func TestPolicyGetListAndAllPolicies(t *testing.T) {
	p1, _ := New(org, "policy-list-one")
	p1.Save()
	p2, _ := New(org, "policy-list-two")
	p2.Save()

	list, err := GetList(org)
	if err != nil {
		t.Fatalf("GetList failed: %v", err)
	}
	if !stringInSlice("policy-list-one", list) || !stringInSlice("policy-list-two", list) {
		t.Errorf("expected both policies in list, got %v", list)
	}

	all, err := AllPolicies(org)
	if err != nil {
		t.Fatalf("AllPolicies failed: %v", err)
	}
	foundOne, foundTwo := false, false
	for _, p := range all {
		if p.Name == "policy-list-one" {
			foundOne = true
		}
		if p.Name == "policy-list-two" {
			foundTwo = true
		}
	}
	if !foundOne || !foundTwo {
		t.Errorf("expected both policies in AllPolicies")
	}
}

func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestPolicyDelete(t *testing.T) {
	polName := "policy-delete-test"
	p, _ := New(org, polName)
	p.Save()

	pr, _ := p.NewPolicyRevisionFromJSON(revJSONFor("rev-delete-1"))
	pr.Save()

	pgName := "pg-delete-cascade"
	pg, _ := NewPolicyGroup(org, pgName)
	pg.AddPolicy(pr)
	pg.Save()

	if err := p.Delete(); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if found, _ := DoesPolicyExist(org, polName); found {
		t.Errorf("expected policy %s to be deleted", polName)
	}

	pg2, _ := GetPolicyGroup(org, pgName)
	if pg2.NumPolicies() != 0 {
		t.Errorf("expected policy to be removed from group after delete, got %d", pg2.NumPolicies())
	}
}

func TestPolicyRevisionResponse(t *testing.T) {
	polName := "policy-revision-response"
	p, _ := New(org, polName)
	p.Save()

	revID1 := "rev-response-1"
	revID2 := "rev-response-2"
	pr1, _ := p.NewPolicyRevisionFromJSON(revJSONFor(revID1))
	pr1.Save()
	pr2, _ := p.NewPolicyRevisionFromJSON(revJSONFor(revID2))
	pr2.Save()

	resp := p.RevisionResponse()
	if _, ok := resp[revID1]; !ok {
		t.Errorf("expected revision %s in response", revID1)
	}
	if _, ok := resp[revID2]; !ok {
		t.Errorf("expected revision %s in response", revID2)
	}
}

func TestPolicyMostRecentRevision(t *testing.T) {
	polName := "policy-most-recent"
	p, _ := New(org, polName)
	p.Save()

	revID1 := "rev-recent-1"
	revID2 := "rev-recent-2"
	pr1, _ := p.NewPolicyRevisionFromJSON(revJSONFor(revID1))
	pr1.Save()
	pr2, _ := p.NewPolicyRevisionFromJSON(revJSONFor(revID2))
	pr2.Save()

	recent, err := p.MostRecentRevision()
	if err != nil {
		t.Fatalf("MostRecentRevision failed: %v", err)
	}
	if recent == nil {
		t.Fatal("expected a revision")
	}
	if recent.RevisionId != revID1 && recent.RevisionId != revID2 {
		t.Errorf("expected one of the two revisions, got %s", recent.RevisionId)
	}
}

func TestPolicyRevisionDelete(t *testing.T) {
	polName := "policy-rev-delete"
	p, _ := New(org, polName)
	p.Save()

	revID := "rev-to-delete"
	pr, _ := p.NewPolicyRevisionFromJSON(revJSONFor(revID))
	pr.Save()

	if _, err := p.GetPolicyRevision(revID); err != nil {
		t.Fatalf("GetPolicyRevision failed before delete: %v", err)
	}

	if err := pr.Delete(); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := p.GetPolicyRevision(revID); err == nil {
		t.Error("expected error after deleting revision")
	}
}

func TestPolicyGroupGetAllAndExists(t *testing.T) {
	pgName1 := "pg-all-one"
	pgName2 := "pg-all-two"
	pg1, _ := NewPolicyGroup(org, pgName1)
	pg1.Save()
	pg2, _ := NewPolicyGroup(org, pgName2)
	pg2.Save()

	found, err := DoesPolicyGroupExist(org, pgName1)
	if err != nil {
		t.Fatalf("DoesPolicyGroupExist failed: %v", err)
	}
	if !found {
		t.Errorf("expected policy group %s to exist", pgName1)
	}

	found, _ = DoesPolicyGroupExist(org, "pg-all-missing")
	if found {
		t.Error("expected missing policy group not to exist")
	}

	all, err := GetAllPolicyGroups(org)
	if err != nil {
		t.Fatalf("GetAllPolicyGroups failed: %v", err)
	}
	foundOne, foundTwo := false, false
	for _, pg := range all {
		if pg.Name == pgName1 {
			foundOne = true
		}
		if pg.Name == pgName2 {
			foundTwo = true
		}
	}
	if !foundOne || !foundTwo {
		t.Errorf("expected both groups in GetAllPolicyGroups")
	}
}

func TestPolicyGroupContainsAndChecks(t *testing.T) {
	pgName := "pg-contains"
	polName := "policy-contains-test"
	revID := "rev-contains-1"

	p, _ := New(org, polName)
	p.Save()
	pr, _ := p.NewPolicyRevisionFromJSON(revJSONFor(revID))
	pr.Save()

	pg, _ := NewPolicyGroup(org, pgName)
	pg.AddPolicy(pr)
	pg.Save()

	contains, err := pg.DoesContainPolicy(org, polName)
	if err != nil {
		t.Fatalf("DoesContainPolicy failed: %v", err)
	}
	if !contains {
		t.Error("expected group to contain policy")
	}

	if !pg.CheckPolicyAndRevision(polName, revID) {
		t.Error("expected CheckPolicyAndRevision to match")
	}
	if pg.CheckPolicyAndRevision(polName, "wrong-rev") {
		t.Error("expected CheckPolicyAndRevision to fail for wrong revision")
	}

	pm := pg.GetPolicyMap()
	if pm[polName]["revision_id"] != revID {
		t.Errorf("expected revision id %s in policy map", revID)
	}
}

func TestPolicyGroupRemoveByRevision(t *testing.T) {
	pgName := "pg-remove-by-rev"
	polName := "policy-remove-by-rev"
	revID := "rev-remove-by-rev-1"

	p, _ := New(org, polName)
	p.Save()
	pr, _ := p.NewPolicyRevisionFromJSON(revJSONFor(revID))
	pr.Save()

	pg, _ := NewPolicyGroup(org, pgName)
	pg.AddPolicy(pr)
	pg.Save()

	if err := pg.removePolicyByRevision(polName, "wrong-rev"); err == nil {
		t.Error("expected error when removing wrong revision")
	}

	if err := pg.removePolicyByRevision(polName, revID); err != nil {
		t.Fatalf("removePolicyByRevision failed: %v", err)
	}
	if pg.NumPolicies() != 0 {
		t.Errorf("expected 0 policies after removal, got %d", pg.NumPolicies())
	}
}
