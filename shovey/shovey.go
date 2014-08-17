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
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/serfin"
	"github.com/codeskyblue/go-uuid"
	"github.com/ctdk/goiardi/util"
	serfclient "github.com/hashicorp/serf/client"
	"math"
	"net/http"
	"regexp"
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
}

type ShoveyRun struct {
	ID int
	ShoveyUUID string 
	NodeName string
	Status string
	AckTime time.Time
	EndTime time.Time
	Output string
	ErrMsg string
	ExitStatus uint8
}

type Qerror interface {
	String() string
	Error() string
	Status() string
	SetStatus(string)
	UpNodes() []string
	DownNodes() []string
	SetUpNodes([]string)
	SetDownNodes([]string)
}

type qerror struct {
	msg string
	status string
	upNodes []string
	downNodes []string
}

func newQuerror(text string) Qerror {
	return &qerror{msg: text,
		status: "job_failed",
	}
}

// Errorf creates a new Qerror, with a formatted error string.
func Errorf(format string, a ...interface{}) Qerror {
	return newQuerror(fmt.Sprintf(format, a...))
}

// Error returns the Qerror error message.
func (e *qerror) Error() string {
	return e.msg
}

// CastErr will easily cast a different kind of error to a Qerror.
func CastErr(err error) Qerror {
	return Errorf(err.Error())
}

func (e *qerror) String() string {
	return e.msg
}

// Set the Qerror status string.
func (e *qerror) SetStatus(s string) {
	e.status = s
}

// Returns the Qerror's HTTP status code.
func (e *qerror) Status() string {
	return e.status
}

func (e *qerror) UpNodes() []string {
	return e.upNodes
}

func (e *qerror) DownNodes() []string {
	return e.downNodes
}

func (e *qerror) SetUpNodes(u []string) {
	e.upNodes = u
}

func (e *qerror) SetDownNodes(d []string) {
	e.downNodes = d
}

func New(command string, timeout int, quorumStr string, nodeNames []string) (*Shovey, util.Gerror) {
	runID := uuid.New()
	s := &Shovey{ RunID: runID, NodeNames: nodeNames, Command: command, Timeout: time.Duration(timeout) * time.Second, Quorum: quorumStr, Status: "submitted" }
	if config.UsingDB() {
		
	}
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	ds := datastore.New()
	ds.Set("shovey", runID, s)

	err := s.startJobs()
	if err != nil {
		s.Status = err.Status()
		s.save()
		return nil, util.CastErr(err)
	}
	s.Status = "running"
	s.save()

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

func (sr *ShoveyRun) save() util.Gerror {
	if config.UsingDB() {

	}
	ds := datastore.New()
	ds.Set("shovey_run", sr.ShoveyUUID + sr.NodeName, sr)
	return nil
}

func (s *Shovey) GetRun(nodeName string) (*ShoveyRun, util.Gerror) {
	if config.UsingDB() {

	}
	var shoveyRun *ShoveyRun
	ds := datastore.New()
	sr, found := ds.Get("shovey_run", s.RunID + nodeName)
	if !found {
		err := util.Errorf("run %s for node %s not found", s.RunID, nodeName)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	if sr != nil {
		shoveyRun = sr.(*ShoveyRun)
	}
	return shoveyRun, nil
}

func (s *Shovey) GetNodeRuns() ([]*ShoveyRun, util.Gerror) {
	if config.UsingDB() {

	}
	var runs []*ShoveyRun
	for _, n := range s.NodeNames {
		sr, err := s.GetRun(n)
		if err != nil {
			if err.Status() != http.StatusNotFound {
				return nil, err
			}
		} else {
			runs = append(runs, sr)
		}
	}
	return runs, nil
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

func (s *Shovey) Cancel() util.Gerror {
	err := s.CancelRuns(s.NodeNames)
	if err != nil {
		return err
	}
	s.Status = "cancelled"
	err = s.save()
	if err != nil {
		return err
	}

	return nil
}

func (s *Shovey) CancelRuns(nodeNames []string) util.Gerror {
	if config.UsingDB() {

	}
	for _, n := range nodeNames {
		sr, err := s.GetRun(n)
		if err != nil {
			return err
		}
		if sr.Status != "invalid" && sr.Status != "completed" && sr.Status != "failed" && sr.Status != "down" && sr.Status != "nacked" {
			sr.EndTime = time.Now()
			sr.Status = "cancelled"
			err = sr.save()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Shovey) startJobs() Qerror {
	// determine if we meet the quorum
	// First is this a percentage or absolute quorum
	qnum, err := getQuorum(s.Quorum, len(s.NodeNames))
	if err != nil {
		return err
	}
	// query node statuses to see if enough are up
	upNodes, nerr := node.GetNodesByStatus(s.NodeNames, "up")
	if nerr != nil {
		return CastErr(nerr)
	}
	if len(upNodes) < qnum {
		err = Errorf("Not enough nodes were up to execute job %s - got %d, needed at least %d", s.RunID, len(upNodes), qnum)
		err.SetStatus("quorum_failed")
		// be setting up/down nodes here too
		return err
	}

	// if that all worked, send the commands
	errch := make(chan error, 1)
	go func() {
		tagNodes := make([]string, len(upNodes))
		d := make(map[string]bool)
		for i, n := range upNodes {
			tagNodes[i] = n.Name
			d[n.Name] = true
			sr := &ShoveyRun{ ShoveyUUID: s.RunID, NodeName: n.Name, Status: "created" }
			sr.save()
		}
		for _, n := range s.NodeNames {
			if !d[n] {
				sr := &ShoveyRun{ ShoveyUUID: s.RunID, NodeName: n, Status: "down", EndTime: time.Now() }
				sr.save()
			}
		}
		// make sure this is the right amount of buffering
		payload := make(map[string]string)
		payload["run_id"] = s.RunID
		payload["command"] = s.Command
		payload["action"] = "start"
		jsonPayload, _ := json.Marshal(payload)
		ackCh := make(chan string, len(tagNodes))
		respCh := make(chan serfclient.NodeResponse, len(tagNodes))
		q := &serfclient.QueryParam{ Name: "shovey", Payload: jsonPayload, FilterNodes: tagNodes, RequestAck: true, AckCh: ackCh, RespCh: respCh }
		qerr := serfin.Serfer.Query(q)
		if qerr != nil {
			errch <- qerr
			return
		}
		errch <- nil
		srCh := make(chan *ShoveyRun, len(upNodes) * 2);

		go func() {
			for sr := range srCh {
				sr.save()
			}
		}()

		for i := 0; i < len(upNodes) * 2; i++{
			select {
			case a := <-ackCh:
				if a == "" {
					continue
				}
				sr, err := s.GetRun(a)
				if err != nil {
					logger.Debugf("err with sr %s: %s", a, err.Error())
					continue
				}
				sr.AckTime = time.Now()
				srCh <- sr
			case r := <-respCh:
				logger.Debugf("got a response: %v", r)
				break
			case <- time.After(s.Timeout):
				logger.Debugf("timed out, might not be appropriate")
				break
			}
		}
		close(srCh)

		logger.Debugf("out of for/select loop for shovey responses")
	}()
	grerr := <-errch
	if grerr != nil {
		return CastErr(grerr)
	}

	return nil
}

func (s *Shovey) checkCompleted() {
	if config.UsingDB() {

	}
	srs, err := s.GetNodeRuns()
	if err != nil {
		logger.Debugf("Something went wrong checking for job completion: %s", err.Error())
		return
	}
	c := 0
	for _, sr := range srs {
		if sr.Status == "invalid" || sr.Status == "completed" || sr.Status == "failed" || sr.Status == "down" || sr.Status == "nacked" {
			c++
		}
	}
	if c == len(s.NodeNames) {
		s.Status = "completed"
		s.save()
	}
}

func (s *Shovey) ToJSON() (map[string]interface{}, util.Gerror) {
	toJSON := make(map[string]interface{})
	toJSON["id"] = s.RunID
	toJSON["command"] = s.Command
	toJSON["run_timeout"] = s.Timeout
	toJSON["status"] = s.Status
	toJSON["created_at"] = s.CreatedAt
	toJSON["updated_at"] = s.UpdatedAt
	tjnodes := make(map[string][]string)
	
	// we can totally do this more efficiently in SQL mode. Do so when we're
	// done with in-mem mode
	srs, err := s.GetNodeRuns()
	if err != nil {
		return nil, err
	}
	for _, sr := range srs {
		tjnodes[sr.Status] = append(tjnodes[sr.Status], sr.NodeName)
	}
	toJSON["nodes"] = tjnodes

	return toJSON, nil
}

func AllShoveyIDs() ([]string, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	list := ds.GetList("shovey")
	return list, nil
}

func GetList() []string {
	list, _ := AllShoveyIDs()
	return list
}

func (sj *ShoveyRun) UpdateFromJSON(sjData map[string]interface{}) util.Gerror {
	if status, ok := sjData["status"].(string); ok {
		if status == "invalid" || status == "completed" || status == "failed" || status == "nacked" {
			sj.EndTime = time.Now()
		}
		sj.Status = status
	} else {
		logger.Errorf("status isn't getting set?? type: %T status %v", sjData["status"], sjData["status"])
	}
	if output, ok := sjData["output"].(string); ok {
		sj.Output = output
	}
	if errMsg, ok := sjData["err_msg"].(string); ok {
		sj.ErrMsg = errMsg
	}
	if exitStatus, ok := sjData["exit_status"].(float64); ok {
		sj.ExitStatus = uint8(exitStatus)
	}

	err := sj.save()
	if err != nil {
		return err
	}
	//go sj.cleanup()
	go sj.notifyParent()
	return nil
}

func (sj *ShoveyRun) cleanup() {
	payload := make(map[string]string)
	payload["run_id"] = sj.ShoveyUUID
	payload["action"] = "reap"
	jsonPayload, _ := json.Marshal(payload)
	respCh := make(chan serfclient.NodeResponse, 1)
	q := &serfclient.QueryParam{ Name: "shovey", Payload: jsonPayload, FilterNodes: []string{ sj.NodeName }, RespCh: respCh }
	// try 10 times to clean this up, log if it still doesn't work
	for i := 0; i < 10; i++ {
		time.Sleep(250 * time.Millisecond)
		qerr := serfin.Serfer.Query(q)
		if qerr != nil {
			logger.Errorf(qerr.Error())
			return
		}
		response := <-respCh
		logger.Debugf("got response for reaping: %s", string(response.Payload))
		if string(response.Payload) == "success" {
			logger.Debugf("reaped job %s on node %s", sj.ShoveyUUID, string(response.Payload))
			return
		}
	}
	logger.Errorf("Unable to reap job %s on node %s", sj.ShoveyUUID, sj.NodeName)
	return
}

func (sj *ShoveyRun) notifyParent() {
	s, _ := Get(sj.ShoveyUUID)
	s.checkCompleted()
}

func getQuorum(quorum string, numNodes int) (int, Qerror) {
	var qnum float64

	if numNodes == 0 {
		err := Errorf("There's no nodes to make a quorum")
		err.SetStatus("quorum_failed")
		return 0, err
	}

	m := regexp.MustCompile(`^(\d+\.?\d?)%$`)
	z := m.FindStringSubmatch(quorum)
	if z != nil {
		q, err := strconv.ParseFloat(z[1], 64)
		if err != nil {
			qerr := CastErr(err)
			return 0, qerr
		}
		qnum = math.Ceil((q / 100.0) * float64(numNodes))
	} else {
		var err error
		qnum, err = strconv.ParseFloat(quorum, 64)
		if err != nil {
			return 0, CastErr(err)
		}
		if qnum > float64(numNodes) {
			err := Errorf("%d nodes were required for the quorum, but only %d matched the criteria given", qnum, numNodes)
			err.SetStatus("quorum_failed")
			return 0, err
		}
	}

	return int(qnum), nil
}
