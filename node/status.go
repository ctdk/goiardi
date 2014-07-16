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
	"fmt"
)

type NodeStatus struct {
	Node *Node
	Status string
	UpdatedAt time.Time
}

func (n *Node) UpdateStatus(status string) error {
	if status != "new" && status != "up" && status != "down" {
		err := fmt.Errorf("invalid node status %s", status)
		return err
	}
	s := &NodeStatus{ Node: n, Status: status }
	if config.UsingDB() {
		return s.updateNodeStatusSQL()
	}
	s.UpdatedAt = time.Now()
	ds := datastore.New()
	return ds.SetNodeStatus(n.Name, s)
}

func (n *Node)LatestStatus() (*NodeStatus, error) {
	if config.UsingDB() {
		return n.latestStatusSQL()
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
		return n.allStatusesSQL()
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

func (n *Node)deleteStatuses() error {
	if config.UsingDB() {
		err := fmt.Errorf("not needed in SQL mode - foreign keys handle deleting node statuses")
		return err
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

func UnseenNodes() ([]*Node, error){
	if config.UsingDB() {
		return unseenNodesSQL()
	}
	var downNodes []*Node
	nodes := AllNodes()
	t := time.Now().Add(-10 * time.Minute)
	for _, n := range nodes {
		ns, _ := n.LatestStatus()
		if ns == nil {
			continue
		}
		if ns.UpdatedAt.Before(t) {
			downNodes = append(downNodes, n)
		}
	}
	return downNodes, nil
}
