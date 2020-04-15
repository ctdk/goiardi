/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Package shovey provides a means for pushing jobs out to nodes to be run
// independently of a chef-client run.
package shovey

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ctdk/chefcrypto"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/secret"
	"github.com/ctdk/goiardi/serfin"
	"github.com/ctdk/goiardi/util"
	serfclient "github.com/hashicorp/serf/client"
	"github.com/pborman/uuid"
	"github.com/tideland/golib/logger"
)

// Shovey holds all the overall information for a shovey run common to all nodes
// running the command.
type Shovey struct {
	RunID     string        `json:"id"`
	NodeNames []string      `json:"nodes"`
	Command   string        `json:"command"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Status    string        `json:"status"`
	Timeout   time.Duration `json:"timeout"`
	Quorum    string        `json:"quorum"`
	org       *organization.Organization
}

// ShoveyRun represents a node's shovey run.
type ShoveyRun struct {
	ID         int       `json:"-"`
	ShoveyUUID string    `json:"run_id"`
	NodeName   string    `json:"node_name"`
	OrgNodeID  string    `json:"org_node_id"`
	Status     string    `json:"status"`
	AckTime    time.Time `json:"ack_time"`
	EndTime    time.Time `json:"end_time"`
	Error      string    `json:"error"`
	ExitStatus uint8     `json:"exit_status"`
	org        *organization.Organization
}

// ShoveyRunStream holds a chunk of output from a shovey run.
type ShoveyRunStream struct {
	ShoveyUUID string
	NodeName   string
	OrgNodeID  string
	Seq        int
	OutputType string
	Output     string
	IsLast     bool
	CreatedAt  time.Time
	org        *organization.Organization
}

// BySeq is a type used to sort ShoveyRunStreams.
type BySeq []*ShoveyRunStream

// Qerror is a special error type for shovey runs.
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
	msg       string
	status    string
	upNodes   []string
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

// String returns a string representation of a Qerror.
func (e *qerror) String() string {
	return e.msg
}

// Set the Qerror status string.
func (e *qerror) SetStatus(s string) {
	e.status = s
}

// Status returns the Qerror's HTTP status code.
func (e *qerror) Status() string {
	return e.status
}

// UpNodes returns the nodes known to be up by this Qerror.
func (e *qerror) UpNodes() []string {
	return e.upNodes
}

// DownNodes returns the nodes known to be up by this Qerror
func (e *qerror) DownNodes() []string {
	return e.downNodes
}

// SetUpNodes sets the nodes that are up for this Qerror.
func (e *qerror) SetUpNodes(u []string) {
	e.upNodes = u
}

// SetDownNodes sets the nodes that are down for this Qerror.
func (e *qerror) SetDownNodes(d []string) {
	e.downNodes = d
}

// New creates a new shovey instance.
func New(org *organization.Organization, command string, timeout int, quorumStr string, nodeNames []string) (*Shovey, util.Gerror) {
	var found bool
	runID := uuid.New()

	// Conflicting uuids are unlikely, but conceivable.
	if config.UsingDB() {
		var err error
		found, err = checkForShoveySQL(org, runID)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("shovey"), runID)
	}

	// unlikely
	if found {
		err := util.Errorf("a shovey run with this run id already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	s := &Shovey{RunID: runID, NodeNames: nodeNames, Command: command, Timeout: time.Duration(timeout), Quorum: quorumStr, Status: "submitted", org: org}

	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()

	err := s.save()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Start kicks off all the shovey runs for this shovey instance.
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
		return s.saveSQL()
	}
	s.UpdatedAt = time.Now()

	ds := datastore.New()
	ds.Set(s.org.DataKey("shovey"), s.RunID, s)

	return nil
}

func (sr *ShoveyRun) save() util.Gerror {
	if config.UsingDB() {
		return sr.saveSQL()
	}
	ds := datastore.New()
	ds.Set(sr.org.DataKey("shovey_run"), sr.ShoveyUUID+sr.NodeName, sr)
	return nil
}

// GetRun gets a particular node's shovey run associated with this shovey
// instance.
func (s *Shovey) GetRun(nodeName string) (*ShoveyRun, util.Gerror) {
	if config.UsingDB() {
		return s.getShoveyRunSQL(nodeName)
	}
	var shoveyRun *ShoveyRun
	ds := datastore.New()
	sr, found := ds.Get(s.org.DataKey("shovey_run"), s.RunID+nodeName)
	if !found {
		err := util.Errorf("run %s for node %s not found", s.RunID, nodeName)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	if sr != nil {
		shoveyRun = sr.(*ShoveyRun)
		shoveyRun.org = s.org
	}
	return shoveyRun, nil
}

// GetNodeRuns gets all of the ShoveyRuns associated with this shovey instance.
func (s *Shovey) GetNodeRuns() ([]*ShoveyRun, util.Gerror) {
	if config.UsingDB() {
		return s.getShoveyNodeRunsSQL()
	}
	runs := make([]*ShoveyRun, 0, len(s.NodeNames))
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

// Get a shovey instance with the given run id.
func Get(org *organization.Organization, runID string) (*Shovey, util.Gerror) {
	if config.UsingDB() {
		return getShoveySQL(org, runID)
	}
	var shove *Shovey
	ds := datastore.New()
	s, found := ds.Get(org.DataKey("shovey"), runID)
	if s != nil {
		shove = s.(*Shovey)
		shove.org = org
	}
	if !found {
		err := util.Errorf("shovey job %s not found", runID)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return shove, nil
}

// DoesExist checks if there is a shovey instance with the given run id.
func DoesExist(org *organization.Organization, runID string) (bool, util.Gerror) {
	if config.UsingDB() {
		found, err := checkForShoveySQL(org, runID)
		if err != nil {
			serr := util.CastErr(err)
			return false, serr
		}
		return found, nil
	}
	ds := datastore.New()
	_, found := ds.Get(org.DataKey("shovey"), runID)
	return found, nil
}

// Cancel cancels all ShoveyRuns associated with this shovey instance.
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

// CancelRuns cancels the shovey runs given in the slice of strings with the
// node names to cancel jobs on.
func (s *Shovey) CancelRuns(nodeNames []string) util.Gerror {
	if config.UsingDB() {
		err := s.cancelRunsSQL()
		if err != nil {
			return err
		}
	} else {
		for _, n := range nodeNames {
			sr, err := s.GetRun(n)
			if err != nil {
				return err
			}
			if sr.Status != "invalid" && sr.Status != "succeeded" && sr.Status != "failed" && sr.Status != "down" && sr.Status != "nacked" {
				sr.EndTime = time.Now()
				sr.Status = "cancelled"
				err = sr.save()
				if err != nil {
					return err
				}
			}
		}
	}
	if len(nodeNames) == len(s.NodeNames) {
		sort.Strings(nodeNames)
		sort.Strings(s.NodeNames)
		if reflect.DeepEqual(nodeNames, s.NodeNames) {
			s.Status = "cancelled"
			s.save()
		}
	} else {
		s.checkCompleted()
	}

	payload := make(map[string]string)
	payload["action"] = "cancel"
	payload["run_id"] = s.RunID
	payload["time"] = time.Now().Format(time.RFC3339)
	sig, serr := s.signRequest(payload)
	if serr != nil {
		return util.CastErr(serr)
	}
	payload["signature"] = sig
	jsonPayload, _ := json.Marshal(payload)
	ackCh := make(chan string, len(nodeNames))

	orgNodeIDs := orgNodeNameSlice(s.org, nodeNames)

	q := &serfclient.QueryParam{Name: "shovey", Payload: jsonPayload, FilterNodes: orgNodeIDs, RequestAck: true, AckCh: ackCh}
	errch := make(chan error)
	go serfin.Query(q, errch)
	err := <-errch

	if err != nil {
		return util.CastErr(err)
	}
	doneCh := make(chan struct{})
	go func() {
		for c := range ackCh {
			logger.Debugf("Received acknowledgement from %s", c)
		}
		doneCh <- struct{}{}
	}()
	select {
	case <-doneCh:
		logger.Infof("All nodes acknowledged cancellation")
		// probably do a report here?
	case <-time.After(time.Duration(60) * time.Second):
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
	upNodes, nerr := node.GetNodesByStatus(s.org, s.NodeNames, "up")
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
	errch := make(chan error)
	go func() {
		tagNodes := make([]string, len(upNodes))
		d := make(map[string]bool)
		for i, n := range upNodes {
			tagNodes[i] = n.Name
			d[n.Name] = true
			sr := &ShoveyRun{ShoveyUUID: s.RunID, NodeName: n.Name, OrgNodeID: orgNodeName(s.org, n.Name), Status: "created"}
			err := sr.save()
			if err != nil {
				logger.Errorf("error saving shovey run: %s", err.Error())
				errch <- err
				return
			}
		}
		for _, n := range s.NodeNames {
			if !d[n] {
				sr := &ShoveyRun{ShoveyUUID: s.RunID, NodeName: n, OrgNodeID: orgNodeName(s.org, n), Status: "down", EndTime: time.Now()}
				err := sr.save()
				if err != nil {
					logger.Errorf("error saving shovey run: %s", err.Error())
					errch <- err
					return
				}
			}
		}
		// make sure this is the right amount of buffering
		payload := make(map[string]string)
		payload["run_id"] = s.RunID
		payload["command"] = s.Command
		payload["action"] = "start"
		payload["time"] = time.Now().Format(time.RFC3339)
		payload["timeout"] = fmt.Sprintf("%d", s.Timeout)
		sig, serr := s.signRequest(payload)
		if serr != nil {
			errch <- serr
			return
		}
		payload["signature"] = sig
		jsonPayload, _ := json.Marshal(payload)
		ackCh := make(chan string, len(tagNodes))
		respCh := make(chan serfclient.NodeResponse, len(tagNodes))

		orgNodeIDs := orgNodeNameSlice(s.org, tagNodes)
		q := &serfclient.QueryParam{Name: "shovey", Payload: jsonPayload, FilterNodes: orgNodeIDs, RequestAck: true, AckCh: ackCh, RespCh: respCh}
		qerrch := make(chan error)
		go serfin.Query(q, qerrch)
		qerr := <-qerrch
		if qerr != nil {
			errch <- qerr
			return
		}

		errch <- nil
		srCh := make(chan *ShoveyRun, len(upNodes)*2)

		go func() {
			for sr := range srCh {
				sr.save()
			}
		}()

		for i := 0; i < len(upNodes)*2; i++ {
			select {
			case a := <-ackCh:
				if a == "" {
					continue
				}

				ok, nn := s.extractNodeName(a)
				if !ok {
					continue
				}

				sr, err := s.GetRun(nn)
				if err != nil {
					logger.Debugf("err with sr %s: %s", a, err.Error())
					continue
				}
				sr.AckTime = time.Now()
				srCh <- sr
			case r := <-respCh:
				logger.Debugf("got a response: %v", r)
				break
			case <-time.After(s.Timeout * time.Second):
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
		s.checkCompletedSQL()
		return
	}
	srs, err := s.GetNodeRuns()
	if err != nil {
		logger.Debugf("Something went wrong checking for job completion: %s", err.Error())
		return
	}
	c := 0
	for _, sr := range srs {
		if sr.Status == "invalid" || sr.Status == "succeeded" || sr.Status == "failed" || sr.Status == "down" || sr.Status == "nacked" || sr.Status == "cancelled" {
			c++
		}
	}
	if c == len(s.NodeNames) {
		s.Status = "complete"
		s.save()
	}
}

// ToJSON formats a shovey instance to render as JSON for the client.
func (s *Shovey) ToJSON() (map[string]interface{}, util.Gerror) {
	toJSON := make(map[string]interface{})
	toJSON["id"] = s.RunID
	toJSON["command"] = s.Command
	toJSON["run_timeout"] = s.Timeout
	toJSON["status"] = s.Status
	toJSON["created_at"] = s.CreatedAt
	toJSON["updated_at"] = s.UpdatedAt
	toJSON["organization_id"] = s.org.GetId()
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

// AllShoveyIDs returns all shovey run ids.
func AllShoveyIDs(org *organization.Organization) ([]string, util.Gerror) {
	if config.UsingDB() {
		return allShoveyIDsSQL(org)
	}
	ds := datastore.New()
	list := ds.GetList(org.DataKey("shovey"))
	return list, nil
}

// GetList returns a list of all shovey ids.
func GetList(org *organization.Organization) []string {
	list, _ := AllShoveyIDs(org)
	return list
}

// AllShoveys returns all shovey objects on the server
func AllShoveys(org *organization.Organization) []*Shovey {
	var shoveys []*Shovey
	if config.UsingDB() {
		return allShoveysSQL(org)
	}
	shoveList := GetList(org)
	for _, s := range shoveList {
		sh, err := Get(org, s)
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		shoveys = append(shoveys, sh)
	}

	return shoveys
}

func AllShoveyRuns(org *organization.Organization) []*ShoveyRun {
	var shoveyRuns []*ShoveyRun
	shoveys := AllShoveys(org)
	for _, s := range shoveys {
		runs, err := s.GetNodeRuns()
		s := make([]*ShoveyRun, 0, len(shoveyRuns)+len(runs))
		copy(s, shoveyRuns)
		shoveyRuns = s
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		shoveyRuns = append(shoveyRuns, runs...)
	}
	return shoveyRuns
}

func AllShoveyRunStreams(org *organization.Organization) []*ShoveyRunStream {
	var streams []*ShoveyRunStream
	shoveyRuns := AllShoveyRuns(org)
	outputTypes := []string{"stdout", "stderr"}
	for _, sr := range shoveyRuns {
		for _, t := range outputTypes {
			srs, err := sr.GetStreamOutput(t, 0)
			s := make([]*ShoveyRunStream, 0, len(streams)+len(srs))
			copy(s, streams)
			streams = s
			if err != nil {
				logger.Criticalf(err.Error())
				os.Exit(1)
			}
			streams = append(streams, srs...)
		}
	}
	return streams
}

// UpdateFromJSON updates a ShoveyRun with the given JSON from the client.
func (sr *ShoveyRun) UpdateFromJSON(srData map[string]interface{}) util.Gerror {
	if status, ok := srData["status"].(string); ok {
		if status == "invalid" || status == "succeeded" || status == "failed" || status == "nacked" {
			sr.EndTime = time.Now()
		}
		sr.Status = status
	} else {
		logger.Errorf("status isn't getting set?? type: %T status %v", srData["status"], srData["status"])
	}
	if errorStr, ok := srData["error"].(string); ok {
		sr.Error = errorStr
	}
	if exitStatus, ok := intify(srData["exit_status"]); ok {
		sr.ExitStatus = uint8(exitStatus)
	}

	err := sr.save()
	if err != nil {
		return err
	}
	go sr.notifyParent()
	return nil
}

func (sr *ShoveyRun) notifyParent() {
	s, _ := Get(sr.org, sr.ShoveyUUID)
	s.checkCompleted()
}

// AddStreamOutput adds a chunk of output from the job to the output list on the
// server stored in the ShoveyRunStream objects.
func (sr *ShoveyRun) AddStreamOutput(output string, outputType string, seq int, isLast bool) util.Gerror {
	if config.UsingDB() {
		return sr.addStreamOutSQL(output, outputType, seq, isLast)
	}
	stream := &ShoveyRunStream{ShoveyUUID: sr.ShoveyUUID, NodeName: sr.NodeName, Seq: seq, OutputType: outputType, Output: output, IsLast: isLast, CreatedAt: time.Now(), org: sr.org}
	ds := datastore.New()
	streamKey := fmt.Sprintf("%s_%s_%s_%d", sr.ShoveyUUID, sr.NodeName, outputType, seq)
	logger.Debugf("Setting %s", streamKey)
	_, found := ds.Get(sr.org.DataKey("shovey_run_stream"), streamKey)
	if found {
		err := util.Errorf("sequence %d for %s - %s already exists", seq, sr.ShoveyUUID, sr.NodeName)
		err.SetStatus(http.StatusConflict)
		return err
	}
	ds.Set("shovey_run_stream", streamKey, stream)

	return nil
}

// GetStreamOutput gets all ShoveyRunStream objects associated with a ShoveyRun
// of the given output type.
func (sr *ShoveyRun) GetStreamOutput(outputType string, seq int) ([]*ShoveyRunStream, util.Gerror) {
	if config.UsingDB() {
		return sr.getStreamOutSQL(outputType, seq)
	}
	var streams []*ShoveyRunStream
	ds := datastore.New()
	for i := seq; ; i++ {
		logger.Debugf("Getting %s", fmt.Sprintf("%s_%s_%s_%d", sr.ShoveyUUID, sr.NodeName, outputType, i))
		s, found := ds.Get(sr.org.DataKey("shovey_run_stream"), fmt.Sprintf("%s_%s_%s_%d", sr.ShoveyUUID, sr.NodeName, outputType, i))
		s.(*ShoveyRunStream).org = sr.org
		if !found {
			break
		}
		logger.Debugf("got a stream: %v", s)
		streams = append(streams, s.(*ShoveyRunStream))
	}
	return streams, nil
}

// CombineStreamOutput combines a ShoveyRun's output streams.
func (sr *ShoveyRun) CombineStreamOutput(outputType string, seq int) (string, util.Gerror) {
	// TODO: This could probably be all SQLized and made way simpler when
	// using a database. Do see.
	stream, err := sr.GetStreamOutput(outputType, seq)
	if err != nil {
		return "", err
	}
	sort.Sort(BySeq(stream))
	var combinedOutput bytes.Buffer
	for _, sitem := range stream {
		combinedOutput.WriteString(sitem.Output)
	}
	return combinedOutput.String(), nil
}

// ToJSON formats a ShoveyRun for marshalling as JSON.
func (sr *ShoveyRun) ToJSON() (map[string]interface{}, util.Gerror) {
	var err util.Gerror
	toJSON := make(map[string]interface{})
	toJSON["run_id"] = sr.ShoveyUUID
	toJSON["node_name"] = sr.NodeName
	toJSON["status"] = sr.Status
	toJSON["ack_time"] = sr.AckTime
	toJSON["end_time"] = sr.EndTime
	toJSON["error"] = sr.Error
	toJSON["exit_status"] = sr.ExitStatus
	toJSON["output"], err = sr.CombineStreamOutput("stdout", 0)
	if err != nil {
		return nil, err
	}
	toJSON["stderr"], err = sr.CombineStreamOutput("stderr", 0)
	if err != nil {
		return nil, err
	}
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
			err := Errorf("%f nodes were required for the quorum, but only %d matched the criteria given", qnum, numNodes)
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
		parr[u] = util.JoinStr(k, ": ", payload[k])
	}
	payloadBlock := strings.Join(parr, "\n")

	var pk *rsa.PrivateKey
	if config.UsingExternalSecrets() {
		var err error
		pk, err = secret.GetSigningKey(config.Config.VaultShoveyKey)
		if err != nil {
			return "", err
		}
	} else {
		var err error
		if pk, err = s.org.ShoveyPrivKey(); err != nil {
			return "", err
		}
	}
	sig, err := chefcrypto.SignTextBlock(payloadBlock, pk)
	if err != nil {
		return "", err
	}
	return sig, nil
}

// ImportShovey is used to import shovey jobs from the exported JSON dump.
func ImportShovey(org *organization.Organization, shoveyJSON map[string]interface{}) error {
	runID := shoveyJSON["id"].(string)
	nn := shoveyJSON["nodes"].([]interface{})
	nodeNames := make([]string, len(nn))
	for i, v := range nn {
		nodeNames[i] = v.(string)
	}
	command := shoveyJSON["command"].(string)
	ca := shoveyJSON["created_at"].(string)
	createdAt, err := time.Parse(time.RFC3339, ca)
	if err != nil {
		return nil
	}
	ua := shoveyJSON["updated_at"].(string)
	updatedAt, err := time.Parse(time.RFC3339, ua)
	if err != nil {
		return nil
	}
	status := shoveyJSON["status"].(string)
	ttmp, _ := intify(shoveyJSON["timeout"])
	timeout := time.Duration(ttmp)
	quorum := shoveyJSON["quorum"].(string)
	s := &Shovey{RunID: runID, NodeNames: nodeNames, Command: command, CreatedAt: createdAt, UpdatedAt: updatedAt, Status: status, Timeout: timeout, Quorum: quorum, org: org}
	return s.importSave()
}

// ImportShoveyRun is used to import shovey jobs from the exported JSON dump.
func ImportShoveyRun(org *organization.Organization, sRunJSON map[string]interface{}) error {
	shoveyUUID := sRunJSON["run_id"].(string)
	nodeName := sRunJSON["node_name"].(string)
	status := sRunJSON["status"].(string)
	var ackTime, endTime time.Time
	if at, ok := sRunJSON["ack_time"].(string); ok {
		var err error
		if ackTime, err = time.Parse(time.RFC3339, at); err != nil {
			return err
		}
	}
	if et, ok := sRunJSON["end_time"].(string); ok {
		var err error
		if endTime, err = time.Parse(time.RFC3339, et); err != nil {
			return err
		}
	}

	errMsg := sRunJSON["error"].(string)
	extmp, _ := intify(sRunJSON["exit_status"])
	exitStatus := uint8(extmp)
	sr := &ShoveyRun{ShoveyUUID: shoveyUUID, NodeName: nodeName, Status: status, AckTime: ackTime, EndTime: endTime, Error: errMsg, ExitStatus: exitStatus, org: org}

	// This can use the normal save function
	return sr.save()
}

// ImportShoveyRunStream is used to import shovey jobs from the exported JSON
// dump.
func ImportShoveyRunStream(org *organization.Organization, srStreamJSON map[string]interface{}) error {
	shoveyUUID := srStreamJSON["ShoveyUUID"].(string)
	nodeName := srStreamJSON["NodeName"].(string)
	seqtmp, _ := intify(srStreamJSON["Seq"])
	seq := int(seqtmp)
	outputType := srStreamJSON["OutputType"].(string)
	output := srStreamJSON["Output"].(string)
	isLast := srStreamJSON["IsLast"].(bool)
	ca := srStreamJSON["CreatedAt"].(string)
	createdAt, err := time.Parse(time.RFC3339, ca)
	if err != nil {
		return err
	}
	srs := &ShoveyRunStream{ShoveyUUID: shoveyUUID, NodeName: nodeName, Seq: seq, OutputType: outputType, Output: output, IsLast: isLast, CreatedAt: createdAt, org: org}
	return srs.importSave()
}

func (s *Shovey) importSave() error {
	if config.UsingDB() {
		return s.importSaveSQL()
	}
	ds := datastore.New()
	ds.Set(s.org.DataKey("shovey"), s.RunID, s)
	return nil
}

func (srs *ShoveyRunStream) importSave() error {
	if config.UsingDB() {
		return srs.importSaveSQL()
	}
	ds := datastore.New()
	skey := fmt.Sprintf("%s_%s_%s_%d", srs.ShoveyUUID, srs.NodeName, srs.OutputType, srs.Seq)
	ds.Set(srs.org.DataKey("shovey_run_stream"), skey, srs)
	return nil
}

func (s BySeq) Len() int           { return len(s) }
func (s BySeq) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s BySeq) Less(i, j int) bool { return s[i].Seq < s[j].Seq }

func (s *Shovey) GetName() string {
	return s.RunID
}

func (s *Shovey) ContainerType() string {
	return "shoveys"
}

func (s *Shovey) ContainerKind() string {
	return "containers"
}

func (s *Shovey) OrgName() string {
	return s.org.Name
}

func intify(i interface{}) (int64, bool) {
	var retint int64
	var ok bool

	switch i := i.(type) {
	case json.Number:
		j, err := i.Int64()
		if err == nil {
			retint = j
			ok = true
		}
	case float64:
		retint = int64(i)
		ok = true
	}
	return retint, ok
}

// Now that there are potentially multiple orgs sending out shovey jobs, we must
// differentiate between nodes that may have the same name in different orgs.
// This simply joins the org ID and the node name together.
func orgNodeName(org *organization.Organization, nodeName string) string {
	return fmt.Sprintf("%s:%s", org.GetName(), nodeName)
}

// And to go along with the above, a handy dandy function to convert a slice of
// node names to the new format.
// Need to return?
func orgNodeNameSlice(org *organization.Organization, nodeNames []string) []string {
	orgNodeIDs := make([]string, len(nodeNames))
	for i, n := range nodeNames {
		n2 := orgNodeName(org, n)
		orgNodeIDs[i] = n2
	}
	return orgNodeIDs
}

func (s *Shovey) extractNodeName(n string) (bool, string) {
	info := strings.Split(n, ":")

	var good bool
	var nodeName string

	if len(info) == 2 && info[0] == s.org.GetName() {
		good = true
		nodeName = info[1]
	}
	return good, nodeName
}
