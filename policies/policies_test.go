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

package policies

import (
	"encoding/gob"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"testing"
)

var org *organization.Organization

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)

	gob.Register(new(organization.Organization))
	gob.Register(new(Policy))
	gob.Register(new(PolicyRevision))
	gob.Register(new(PolicyGroup))

	org, _ = orgloader.New("default", "lurp")
}

func TestPolicyBasics(t *testing.T) {
	testPol := "test-policy"

	p, err := New(org, testPol)
	if err != nil {
		t.Errorf("error creating new policy: %v", err)
	}

	if err = p.Save(); err != nil {
		t.Errorf("error saving policy: %v", err)
	}

	if _, err = Get(org, testPol); err != nil {
		t.Errorf("error reloading policy: %v", err)
	}

	if pNew, err := New(org, testPol); pNew != nil {
		t.Errorf("creating a new policy named '%s' should have failed, but succeeded.", testPol)
		if err != nil {
			t.Errorf("An error was also returned: %v", err)
		}
	}
}
