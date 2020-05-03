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

func TestPolicyGroupCreation(t *testing.T) {
	pgName := "pg1"
	// using the org created in policies_test.go, yo
	pg, err := NewPolicyGroup(org, pgName)
	if err != nil {
		t.Error(err)
	}
	if err = pg.Save(); err != nil {
		t.Error(err)
	}
	pg2, err := GetPolicyGroup(org, pgName)
	if err != nil {
		t.Errorf("GetPolicyGroup had an error: %v", err)
	} else if pg2 == nil {
		t.Error("Somehow GetPolicyGroup did not return an error, but it still returned a nil valued policy group object")
	}
	if pg2.Name != pg.Name {
		t.Errorf("policy group names did not match: expected %s, but got %s", pg.Name, pg2.Name)
	}
}

func TestPolicyGroupAddPolicy(t *testing.T) {
	pgName := "pg2"
	// grab that one policy and revision set from earlier I guess
	p, err := Get(org, policyName)
	if err != nil {
		t.Error(err)
	}

	pr, err := p.GetPolicyRevision(revRevId)
	if err != nil {
		t.Error(err)
	}

	pg, _ := NewPolicyGroup(org, pgName)

	if err = pg.AddPolicy(pr); err != nil {
		t.Error(err)
	}

	if err := pg.Save(); err != nil {
		t.Errorf("saving a policy group with a policy added failed: %v", err)
	}

	pr2, err := pg.GetPolicy(policyName)
	if err != nil {
		t.Error(err)
	} else if pr2 == nil {
		t.Error("buh. pr2 is nil.")
	}

	if pr2.RevisionId != revRevId {
		t.Errorf("the policy revision returned by pg.GetPolicy was incorrect: revision id should have been '%s', but it was '%s'", revRevId, pr2.RevisionId)
	}
	if pr2.PolicyName() != policyName {
		t.Errorf("the policy revision returned by pg.GetPolicy has the wrong policy name: wanted %s, got %s", policyName, pr2.PolicyName())
	}

	// and test retrieving a policy group with a policy added
	pg2, err := GetPolicyGroup(org, pgName)
	if err != nil {
		t.Errorf("error retrieving a policy group saved with a policy attached: %v", err)
	}
	if pg2.NumPolicies() != pg.NumPolicies() {
		t.Errorf("Mismatch with the number of attached policies with a policy group re-fetched after saving: it should have been %d, but the actual value was %d", pg.NumPolicies(), pg2.NumPolicies())
	}
	if pg.NumPolicies() != 1 {
		t.Errorf("the original policy group has the wrong number of attached policies: should have been 1, but is actually %d", pg.NumPolicies())
	}
	if pg2.NumPolicies() != 1 {
		t.Errorf("the reloade3d policy group has the wrong number of attached policies: should have been 1, but is actually %d", pg2.NumPolicies())
	}
}

func TestPolicyGroupRemovePolicy(t *testing.T) {
	pgName := "pg3"
	p, _ := Get(org, policyName)
	pr, _ := p.GetPolicyRevision(revRevId)
	pg, _ := NewPolicyGroup(org, pgName)

	pg.AddPolicy(pr)
	pg.Save()

	// reload!
	pg, _ = GetPolicyGroup(org, pgName)
	err := pg.RemovePolicy(policyName)
	if err != nil {
		t.Error(err)
	}
	if pg.NumPolicies() != 0 {
		t.Errorf("After removing the only attached policy from a policy group, it still had %d policies attached.", pg.NumPolicies())
	}
	pg.Save()

	// reload again!
	pg, _ = GetPolicyGroup(org, pgName)
	if pg.NumPolicies() != 0 {
		t.Errorf("After removing the only attached policy from a policy group, it still had %d policies attached.", pg.NumPolicies())
	}

	if pr2, err := pg.GetPolicy(policyName); err == nil {
		t.Errorf("a policy group with no policies should have had GetPolicy return an error, but this didn't. Value of the policy revision is '%+#v', error is %v", pr2, err)
	}
}
