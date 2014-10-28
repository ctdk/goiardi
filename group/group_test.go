/*
 * Copyright (c) 2013-2014, Jeremy Bingham (<jbingham@gmail.com>)
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

package group

import (
	"encoding/gob"
	"github.com/ctdk/goiardi/organization"
	"testing"
)

// More group tests will be coming, as

func TestGroupCreation(t *testing.T) {
	gob.Register(new(organization.Organization))
	gob.Register(new(Group))
	org, _ := organization.New("florp", "mlorph normph")
	g, err := New(org, "us0rs")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("group us0rs was unexpectedly nil")
	}
	err = g.Save()
	if err != nil {
		t.Errorf(err.Error())
	}
	g2, err := Get(org, "us0rs")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g2 == nil {
		t.Errorf("refetching group didn't work")
	}
	if g2.Name != g.Name {
		t.Errorf("group names didn't match, expected %s, got %s", g.Name, g2.Name)
	}
}

func TestDefaultGroups(t *testing.T) {
	org, _ := organization.New("florp2", "mlorph normph")
	org.Save()
	MakeDefaultGroups(org)
	g, err := Get(org, "users")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group users")
	}
	g, err = Get(org, "admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group admins")
	}
	g, err = Get(org, "billing-admins")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group billing-admins")
	}
	g, err = Get(org, "clients")
	if err != nil {
		t.Errorf(err.Error())
	}
	if g == nil {
		t.Errorf("failed to get created default group clients")
	}

}
