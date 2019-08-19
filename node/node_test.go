/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Node tests
package node

import (
	"encoding/gob"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"testing"
	"time"
)

var org *organization.Organization

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
}

func TestActionAtADistance(t *testing.T) {
	gob.Register(new(organization.Organization))
	org, _ = orgloader.New("default", "boo")
	org.Save()
	indexer.Initialize(config.Config, org)

	n, _ := New(org, "foo2")
	gob.Register(n)
	n.Normal["foo"] = "bar"
	n.Save()
	n2, _ := Get(org, "foo2")
	if n.Name != n2.Name {
		t.Errorf("Node names should have been the same, but weren't, got %s and %s", n.Name, n2.Name)
	}
	if n.Normal["foo"] != n2.Normal["foo"] {
		t.Errorf("Normal attribute 'foo' should have been equal between the two copies of the node, but weren't.")
	}
	n2.Normal["foo"] = "blerp"
	if n.Normal["foo"] == n2.Normal["foo"] {
		t.Errorf("Normal attribute 'foo' should not have been equal between the two copies of the node, but were.")
	}
	n2.Save()
	n3, _ := Get(org, "foo2")
	if n3.Normal["foo"] != n2.Normal["foo"] {
		t.Errorf("Normal attribute 'foo' should have been equal between the two copies of the node after saving a second time, but weren't.")
	}
}

func TestNodeStatus(t *testing.T) {
	n, _ := New(org, "foo3")
	n.Save()
	z := new(NodeStatus)
	gob.Register(z)
	n.UpdateStatus("up")
	ns, err := n.LatestStatus()
	if err != nil {
		t.Errorf(err.Error())
	}
	if ns == nil {
		t.Errorf("node status was nil!")
	} else if ns.Status != "up" {
		t.Errorf("node status should have been 'up', got %s", ns.Status)
	}
	n.UpdateStatus("up")
	n.UpdateStatus("down")
	nses, err := n.AllStatuses()
	if len(nses) != 3 {
		t.Errorf("AllStatuses should have returned 3, but it returned %d", len(nses))
	}
	err = n.deleteStatuses()
	if err != nil {
		t.Errorf(err.Error())
	}
	nses, _ = n.AllStatuses()
	if len(nses) != 0 {
		t.Errorf("AllStatuses should have returned 0 after calling DeleteStatuses, but instead it returned %d", len(nses))
	}
}

func TestNodeStatusDelete(t *testing.T) {
	// clear out any existing node statuses
	nodes := AllNodes(org)
	ds := datastore.New()
	for _, n := range nodes {
		ds.DeleteNodeStatus(n.Name, org.Name)
	}
	dNode, _ := New(org, "deleting_node")
	dNode.Save()
	now := time.Now()
	day := 24 * time.Hour
	nStats := 28
	for i := nStats; i > 0; i-- {
		t := now.Add(-((time.Duration(i) * day) + (5 * time.Minute)))
		var status string
		switch i % 2 {
		case 0:
			status = "up"
		default:
			status = "down"
		}
		ns := &NodeStatus{Node: dNode, Status: status, UpdatedAt: t}
		ds.SetNodeStatus(dNode.Name, org.Name, ns)
	}

	from := 14 * day
	expected := 15
	del, err := DeleteNodeStatusesByAge(from)
	if err != nil {
		t.Error(err)
	}
	if del != expected {
		t.Errorf("Expected %d deleted statuses, but got %d", expected, del)
	}
	orgs, _ := orgloader.AllOrganizations()
	var an int
	for _, urg := range orgs {
		nses := AllNodeStatuses(urg)
		an += len(nses)
	}

	if an != nStats-expected {
		t.Errorf("expected to have %d statuses left, but there were %d", nStats-del, an)
	}
}
