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

package shovey

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/node"
	"github.com/codeskyblue/go-uuid"
	"github.com/ctdk/goiardi/util"
	"time"
)

type Shovey struct {
	RunID string `json:"id"`
	NodeNames []string `json:"nodes"`
	Command string `json:"command"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status string `json:"status"`
	Timeout time.Duration `json:"timeout"`
	Quorum string `json:"quorum"`
	nodeRuns []*ShoveyRun
	nodes []*Node
}

type ShoveyRun struct {
	ID int
	ShoveyUUID string 
	NodeName string
	Status string
	AckTime time.Time
	EndTime time.Time
}

func New(command string, timeout int, quorumStr string, nodes []*Node) (*Shovey, util.Gerror) {
	runID := uuid.New()
	nodeNames := make([]string, len(nodes))
	for i, n := range nodes {
		nodeNames[i] = n.Name
	}
	s := &Shovey{ RunID: runID, NodeNames: nodeNames, Command: command, Timeout: timeout, Quorum: quorumStr, Status: started }
	if config.Config.UsingDB() {
		
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
}

func Get(runID string) (*Shovey, error) {

}

func Cancel(runID string) error {

}


