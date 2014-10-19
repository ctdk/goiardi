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
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"os"
	"time"
)

// NodeStatus records a node's status at a particular time.
type NodeStatus struct {
	Node      *Node
	Status    string
	UpdatedAt time.Time
}

// UpdateStatus updates a node's current status (up, down, or new).
func (n *Node) UpdateStatus(status string) error {
	if status != "new" && status != "up" && status != "down" {
		err := fmt.Errorf("invalid node status %s", status)
		return err
	}
	s := &NodeStatus{Node: n, Status: status}
	if config.UsingDB() {
		return s.updateNodeStatusSQL()
	}
	var nodeDown bool
	if status == "down" {
		nodeDown = true
	}
	if nodeDown != n.isDown {
		n.isDown = nodeDown
		n.Save()
	}
	s.UpdatedAt = time.Now()
	ds := datastore.New()
	return ds.SetNodeStatus(n.Name, n.org.Name, s)
}

// ImportStatus is used by the import function to import node statuses from the
// exported JSON dump.
func ImportStatus(org *organization.Organization, nodeJSON map[string]interface{}) error {
	n := nodeJSON["Node"].(map[string]interface{})
	status := nodeJSON["Status"].(string)
	ut := nodeJSON["UpdatedAt"].(string)
	updatedAt, err := time.Parse(time.RFC3339, ut)
	if err != nil {
		return err
	}
	nodeP, err := Get(org, n["name"].(string))
	if err != nil {
		return nil
	}
	ns := &NodeStatus{Node: nodeP, Status: status, UpdatedAt: updatedAt}
	if config.UsingDB() {
		return ns.importNodeStatus()
	}
	ds := datastore.New()
	nodeP.Save()
	return ds.SetNodeStatus(nodeP.Name, org.Name, ns)
}

// LatestStatus returns the node's latest status.
func (n *Node) LatestStatus() (*NodeStatus, error) {
	if config.UsingDB() {
		return n.latestStatusSQL()
	}
	ds := datastore.New()
	s, err := ds.LatestNodeStatus(n.Name, n.org.Name)
	if err != nil {
		return nil, err
	}
	ns := s.(*NodeStatus)
	return ns, nil
}

// AllStatuses returns all of the node's status reports to date.
func (n *Node) AllStatuses() ([]*NodeStatus, error) {
	if config.UsingDB() {
		return n.allStatusesSQL()
	}
	ds := datastore.New()
	arr, err := ds.AllNodeStatuses(n.Name, n.org.Name)
	if err != nil {
		return nil, err
	}
	ns := make([]*NodeStatus, len(arr))
	for i, v := range arr {
		ns[i] = v.(*NodeStatus)
	}
	return ns, nil
}

// AllNodeStatuses returns all node status reports from the organization, from 
// all nodes.
func AllNodeStatuses(org *organization.Organization) ([]*NodeStatus) {
	var allStatus []*NodeStatus
	nodes := AllNodes(org)
	for _, n := range nodes {
		ns, err := n.AllStatuses()
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		allStatus = append(allStatus, ns...)
	}
	
	return allStatus
}

func (n *Node) deleteStatuses() error {
	if config.UsingDB() {
		err := fmt.Errorf("not needed in SQL mode - foreign keys handle deleting node statuses")
		return err
	}
	ds := datastore.New()
	return ds.DeleteNodeStatus(n.Name, n.org.Name)
}

// ToJSON formats a node status report for export to JSON.
func (ns *NodeStatus) ToJSON() map[string]string {
	nsmap := make(map[string]string)
	nsmap["node_name"] = ns.Node.Name
	nsmap["status"] = ns.Status
	nsmap["updated_at"] = ns.UpdatedAt.Format(time.RFC3339)
	return nsmap
}

// UnseenNodes returns all nodes that have not sent status reports for a while.
func UnseenNodes(org *organization.Organization) ([]*Node, error) {
	if config.UsingDB() {
		return unseenNodesSQL()
	}
	var downNodes []*Node
	nodes := AllNodes(org)
	t := time.Now().Add(-10 * time.Minute)
	for _, n := range nodes {
		ns, _ := n.LatestStatus()
		if ns == nil || n.isDown {
			continue
		}
		if ns.UpdatedAt.Before(t) {
			downNodes = append(downNodes, n)
		}
	}
	return downNodes, nil
}

// GetNodesByStatus returns the nodes that currently have the given status.
func GetNodesByStatus(org *organization.Organization, nodeNames []string, status string) ([]*Node, error) {
	if config.UsingDB() {
		return getNodesByStatusSQL(nodeNames, status)
	}
	var statNodes []*Node
	nodes := make([]*Node, 0, len(nodeNames))
	for _, name := range nodeNames {
		n, _ := Get(org, name)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	statNodes = make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		ns, _ := n.LatestStatus()
		if ns == nil {
			logger.Infof("No status found at all for node %s, skipping", n.Name)
			continue
		}
		if ns.Status == status {
			statNodes = append(statNodes, n)
		}
	}
	return statNodes, nil
}
