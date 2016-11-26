/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package shovey

import (
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"testing"
)

func init() {
	indexer.Initialize(config.Config)
}

var org *organization.Organization

func TestShoveyCreation(t *testing.T) {
	gob.Register(new(organization.Organization))
	org, _ = organization.New("default", "boo")
	org.Save()
	indexer.Initialize(config.Config)
	nn := new(node.Node)
	ns := new(node.NodeStatus)
	gob.Register(nn)
	gob.Register(ns)
	nodes := make([]*node.Node, 5)
	nodeNames := make([]string, 5)
	for i := 0; i < 5; i++ {
		n, _ := node.New(org, fmt.Sprintf("node-shove-%d", i))
		n.Save()
		err := n.UpdateStatus("up")
		if err != nil {
			t.Errorf(err.Error())
		}
		n.Save()
		nodes[i] = n
		nodeNames[i] = n.Name
	}
	z := new(Shovey)
	zz := new(ShoveyRun)
	gob.Register(z)
	gob.Register(zz)
	s, err := New(org, "/bin/ls", 300, "100%", nodeNames)
	if err != nil {
		t.Errorf(err.Error())
	}
	s2, err := Get(org, s.RunID)
	if err != nil {
		t.Errorf(err.Error())
	}
	if s.RunID != s2.RunID {
		t.Errorf("Run IDs should have been equal, but weren't. Got %s and %s", s.RunID, s2.RunID)
	}
	//err = s.Cancel()
	//if err != nil {
	//	t.Errorf(err.Error())
	//}
}
