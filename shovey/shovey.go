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
	"encoding/json"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/serfin"
	"github.com/codeskyblue/go-uuid"
	"github.com/ctdk/goiardi/util"
	serfclient "github.com/hashicorp/serf/client"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
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

func (s *Shovey) startJobs() error {
	// determine if we meet the quorum
	// First is this a percentage or absolute quorum
	qnum, err := getQuorum(s.Quorum, len(s.Nodes))
	if err != nil {
		return err
	}
	// query node statuses to see if enough are up
	upNodes := node.GetNodesByStatus("up")
	if len(upNodes) < qnum {
		err = fmt.Errorf("Not enough nodes were up to execute job %s - got %d, needed at least %d", s.RunID, len(upNodes), qnum)
	}

	// if that all worked, send the commands
	errch := make(chan error, 1)
	go func() {
		tagNodes := make([]string, len(upNodes))
		for i, n := range upNodes {
			tagNodes[i] = n.Name
			sr := &ShoveyRun{ ShoveyUUID: s.RunID, NodeName: n.Name, Status: "created" }
			sr.save()
		}
		// make sure this is the right amount of buffering
		payload := make(map[string]string)
		payload["run_id"] = s.RunID
		payload["command"] = s.Command
		jsonPayload := json.Marshal(payload)
		ackCh := make(chan string, len(tagNodes))
		respCh := make(chan serfclient.NodeResponse, len(tagNodes))
		q := &serfclient.QueryParam{ Name: "shovey", Payload: jsonPayload, FilterNodes: tagNodes, Timeout: s.Timeout, RequestAck: true, AckCh: ackCh, RespCh: respCh }
		qerr := serfclient.Query(q)
		if qerr != nil {
			errch <- qerr
			return
		}

		for {
			select {
			case a := <-ackCh:

			case r := <-respCh:

			case <- time.After(s.Timeout * time.Second):
				break
			}
		}
	}()
	err <-errch
	if err != nil {
		return err
	}

	return nil
}

func getQuorum(quorum string, numNodes int) (int, error) {
	var qnum float64

	if numNodes == 0 {
		err := fmt.Errorf("There's no nodes to make a quorum")
		return 0, nil
	}

	m := regexp.MustCompile(`^(\d+\.?\d?)%$`)
	z := m.FindStringSubmatch(quorum)
	if z != nil {
		q, err := strconv.ParseFloat(z[1], 64)
		if err != nil {
			return 0, err
		}
		qnum = math.Ceil((q / 100.0) * numNodes)
	} else {
		var err error
		qnum, err = strconv.ParseFloat(quorum, 64)
		if err != nil {
			return 0, err
		}
		if qnum > numNodes {
			err := fmt.Errorf("%d nodes were required for the quorum, but only %d matched the criteria given", qnum, numNodes)
			return 0, err
		}
	}

	return int(qnum), nil
}
