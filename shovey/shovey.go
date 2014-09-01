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
	"github.com/ctdk/goiardi/chefcrypto"
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
	"sort"
	"strconv"
	"strings"
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
	ID int `json:"-"`
	ShoveyUUID string `json:"run_id"`
	NodeName string `json:"node_name"`
	Status string `json:"status"`
	AckTime time.Time `json:"ack_time"`
	EndTime time.Time `json:"end_time"`
	Output string `json:"output"`
	Error string `json:"error"`
	Stderr string `json:"stderr"`
	ExitStatus uint8 `json:"exit_status"`
}

type ShoveyRunStream struct {
	ShoveyUUID string 
	NodeName string
	Seq int
	OutputType string
	Output string
	IsLast bool
	CreatedAt time.Time
}

type BySeq []*ShoveyRunStream

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
	var found bool
	runID := uuid.New()
	
	// Conflicting uuids are unlikely, but conceivable.
	if config.UsingDB() {
		var err error
		found, err = checkForShoveySQL(datastore.Dbh, runID)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get("shovey", runID)
	}

	// unlikely
	if found { 
		err := util.Errorf("a shovey run with this run id already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	s := &Shovey{ RunID: runID, NodeNames: nodeNames, Command: command, Timeout: time.Duration(timeout) * time.Second, Quorum: quorumStr, Status: "submitted" }
	
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	err := s.save()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Shovey) Start() util.Gerror {
	err := s.startJobs()
	if err != nil {
		s.Status = err.Status()
		s.save()
		return util.CastErr(err)
	}
	s.Status = "running"
	s.save()
	return nil
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
	payload := make(map[string]string)
	payload["action"] = "cancel"
	payload["run_id"] = s.RunID
	sig, serr := s.signRequest(payload)
	if serr != nil {
		return util.CastErr(serr)
	}
	payload["signature"] = sig
	jsonPayload, _ := json.Marshal(payload)
	ackCh := make(chan string, len(nodeNames))
	q := &serfclient.QueryParam{ Name: "shovey", Payload: jsonPayload, FilterNodes: nodeNames, RequestAck: true, AckCh: ackCh }
	err := serfin.Serfer.Query(q)
	if err != nil {
		return util.CastErr(err)
	}
	doneCh := make(chan struct{}, 1)
	go func(){
		for c := range ackCh {
			logger.Debugf("Received acknowledgement from %s", c)
		}
		doneCh <- struct{}{}
	}()
	select {
		case <-doneCh:
			logger.Infof("All nodes acknowledged cancellation")
			// probably do a report here?
		case <- time.After(time.Duration(60) * time.Second):
			logger.Errorf("Didn't get all acknowledgements within 60 seconds")
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
		sig, serr := s.signRequest(payload)
		if serr != nil {
			errch <- serr
			return
		}
		payload["signature"] = sig
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
	if errMsg, ok := sjData["stderr"].(string); ok {
		sj.Stderr = errMsg
	}
	if errorStr, ok := sjData["error"].(string); ok {
		sj.Error = errorStr
	}
	if exitStatus, ok := sjData["exit_status"].(float64); ok {
		sj.ExitStatus = uint8(exitStatus)
	}

	err := sj.save()
	if err != nil {
		return err
	}
	go sj.notifyParent()
	return nil
}

func (sj *ShoveyRun) notifyParent() {
	s, _ := Get(sj.ShoveyUUID)
	s.checkCompleted()
}

func (sj *ShoveyRun) AddStreamOutput(output string, outputType string, seq int, isLast bool) util.Gerror {
	if config.UsingDB() {

	}
	stream := &ShoveyRunStream{ ShoveyUUID: sj.ShoveyUUID, NodeName: sj.NodeName, Seq: seq, OutputType: outputType, Output: output, IsLast: isLast, CreatedAt: time.Now() }
	ds := datastore.New()
	streamKey := fmt.Sprintf("%s_%s_%s_%d", sj.ShoveyUUID, sj.NodeName, outputType, seq)
	logger.Debugf("Setting %s", streamKey)
	_, found := ds.Get("shovey_run_stream", streamKey)
	if found {
		err := util.Errorf("sequence %d for %s - %s already exists", seq, sj.ShoveyUUID, sj.NodeName)
		err.SetStatus(http.StatusConflict)
		return err
	}
	ds.Set("shovey_run_stream", streamKey, stream)

	return nil
}

func (sj *ShoveyRun) GetStreamOutput(outputType string, seq int) ([]*ShoveyRunStream, util.Gerror) {
	if config.UsingDB() {

	}
	var streams []*ShoveyRunStream
	ds := datastore.New()
	for i := seq; ; i++ {
		logger.Debugf("Getting %s", fmt.Sprintf("%s_%s_%s_%d", sj.ShoveyUUID, sj.NodeName, outputType, i))
		s, found := ds.Get("shovey_run_stream", fmt.Sprintf("%s_%s_%s_%d", sj.ShoveyUUID, sj.NodeName, outputType, i))
		if !found {
			break
		}
		logger.Debugf("got a stream: %v", s)
		streams = append(streams, s.(*ShoveyRunStream))
	}
	return streams, nil
}

func (sj *ShoveyRun) ToJSON() (map[string]interface{}, error) {
	toJSON := make(map[string]interface{}
	toJSON["run_id"] = sj.ShoveyUUID
	toJSON["node_name"] = sj.NodeName
	return toJSON, nil
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

func (s *Shovey) signRequest(payload map[string]string) (string, error) {
	if payload == nil {
		return "", fmt.Errorf("No payload given to sign!")
	}
	pkeys := make([]string, len(payload))
	i := 0
	for k := range payload {
		pkeys[i] = k
		i++
	}
	sort.Strings(pkeys)
	parr := make([]string, len(pkeys))
	for u, k := range pkeys {
		parr[u] = fmt.Sprintf("%s: %s", k, payload[k])
	}
	payloadBlock := strings.Join(parr, "\n")

	config.Key.RLock()
	defer config.Key.RUnlock()
	sig, err := chefcrypto.SignTextBlock(payloadBlock, config.Key.PrivKey)
	if err != nil {
		return "", err
	}
	return sig, nil
}

func (s BySeq) Len() int { return len(s) }
func (s BySeq) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySeq) Less(i, j int) bool { return s[i].Seq < s[j].Seq }
