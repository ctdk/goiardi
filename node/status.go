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
	"time"
)

type NodeStatus struct {
	Node *Node
	Status string
	UpdatedAt time.Time
}

func (n *Node) UpdateStatus(status string) error {
	s := n.NewStatus()
	s.Status = status
	return s.Save()
}

func (n *Node)NewStatus() (*NodeStatus, error) {
	if config.UsingDB() {

	}

}

func (n *Node)LatestStatus() (*NodeStatus, error) {

}

func (n *Node)AllStatuses() ([]*NodeStatus, error) {

}
