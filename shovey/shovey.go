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
	"net/http"
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
	NodeRuns []*ShoveyRun
	Nodes []*node.Node
}

type ShoveyRun struct {
	ID int
	ShoveyUUID string 
	NodeName string
	Status string
	AckTime time.Time
	EndTime time.Time
}

func New(command string, timeout int, quorumStr string, nodes []*node.Node) (*Shovey, util.Gerror) {
	runID := uuid.New()
	nodeNames := make([]string, len(nodes))
	for i, n := range nodes {
		nodeNames[i] = n.Name
	}
	s := &Shovey{ RunID: runID, NodeNames: nodeNames, Command: command, Timeout: time.Duration(timeout), Quorum: quorumStr, Status: "submitted" }
	if config.UsingDB() {
		
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	ds := datastore.New()
	ds.Set("shovey", runID, s)

	// TODO: send jobs to nodes, try and get quorum

	return s, nil
}

func (s *Shovey) save() util.Gerror {
	if config.UsingDB() {
		
	}
	s.UpdatedAt = time.Now()

	ds := datastore.New()
	ds.Set("shovey", s.RunID, s)

	return nil
}

func Get(runID string) (*Shovey, util.Gerror) {
	if config.UsingDB() {

	}
	var shove *Shovey
	ds := datastore.New()
	s, found := ds.Get("shovey", runID)
	if s != nil {
		shove = s.(*Shovey)
	}
	if !found {
		err := util.Errorf("shovey job %s not found", runID)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return shove, nil
}

func Cancel(runID string) util.Gerror {
	s, err := Get(runID)
	if err != nil {
		return err
	}
	s.Status = "cancelled"
	err = s.save()
	if err != nil {
		return err
	}

	// TODO: cancel jobs on nodes

	return nil
}

func (s *Shovey) dialNodes() error {

	return nil
}
