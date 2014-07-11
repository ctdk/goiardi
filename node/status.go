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

// Structs, functions, and methods to record and report on a node's status.
// Goes with the shovey functions and the serf stuff.

package node

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"time"
)

type NodeStatus struct {
	Node *Node
	Status string
	UpdatedAt time.Time
}

func (n *Node) UpdateStatus(status string) error {
	s := &NodeStatus{ Node: n, Status: status, UpdatedAt: time.Now() }
	if config.UsingDB() {

	}
	ds := datastore.New()
	return ds.SetNodeStatus(n.Name, s)
}

func (n *Node)LatestStatus() (*NodeStatus, error) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	s, err := ds.LatestNodeStatus(n.Name)
	if err != nil {
		return nil, err
	}
	ns := s.(*NodeStatus)
	return ns, nil
}

func (n *Node)AllStatuses() ([]*NodeStatus, error) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	arr, err := ds.AllNodeStatuses(n.Name)
	if err != nil {
		return nil, err
	}
	ns := make([]*NodeStatus, len(arr))
	for i, v := range arr {
		ns[i] = v.(*NodeStatus)
	}
	return ns, nil
}

func (n *Node)DeleteStatuses() error {
	if config.UsingDB() {

	}
	ds := datastore.New()
	return ds.DeleteNodeStatus(n.Name)
}

func (ns *NodeStatus) ToJSON() map[string]string {
	nsmap := make(map[string]string)
	nsmap["node_name"] = ns.Node.Name
	nsmap["status"] = ns.Status
	nsmap["updated_at"] = ns.UpdatedAt.Format(time.RFC3339)
	return nsmap
}
