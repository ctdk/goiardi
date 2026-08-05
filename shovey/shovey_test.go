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

package shovey

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
)

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(ShoveyRunStream))
}

var org *organization.Organization

func TestShoveyCreation(t *testing.T) {
	gob.Register(new(organization.Organization))
	org, _ = orgloader.New("default", "boo")
	org.Save()
	indexer.Initialize(config.Config, org)
	nn := new(node.Node)
	ns := new(node.NodeStatus)
	gob.Register(nn)
	gob.Register(ns)
	nodes := make([]*node.Node, 5)
	nodeNames := make([]string, 5)
	for i := 0; i < 5; i++ {
		n, _ := node.New(org, fmt.Sprintf("node-shove-%d", i))
		n.Save()
		err := n.UpdateStatus("up")
		if err != nil {
			t.Error(err.Error())
		}
		n.Save()
		nodes[i] = n
		nodeNames[i] = n.Name
	}
	z := new(Shovey)
	zz := new(ShoveyRun)
	gob.Register(z)
	gob.Register(zz)
	s, err := New(org, "/bin/ls", 300, "100%", nodeNames)
	if err != nil {
		t.Error(err.Error())
	}
	s2, err := Get(org, s.RunID)
	if err != nil {
		t.Error(err.Error())
	}
	if s.RunID != s2.RunID {
		t.Errorf("Run IDs should have been equal, but weren't. Got %s and %s", s.RunID, s2.RunID)
	}
	//err = s.Cancel()
	//if err != nil {
	//	t.Errorf(err.Error())
	//}
}

func setupShoveyTest(t *testing.T) (*organization.Organization, func()) {
	t.Helper()
	tmpDir, err := ioutil.TempDir("", "shovey-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err.Error())
	}
	config.Config.PolicyRoot = tmpDir
	o, gerr := orgloader.New("shovey-test-"+t.Name(), "Shovey Test")
	if gerr != nil {
		t.Fatalf("failed to create org: %s", gerr.Error())
	}
	if err := o.Save(); err != nil {
		t.Fatalf("failed to save org: %s", err.Error())
	}
	indexer.Initialize(config.Config, o)
	return o, func() { os.RemoveAll(tmpDir) }
}

func TestShoveyDoesExist(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	s, err := New(org, "/bin/ls", 300, "100%", []string{})
	if err != nil {
		t.Fatalf("New failed: %s", err.Error())
	}
	found, err := DoesExist(org, s.RunID)
	if err != nil {
		t.Fatalf("DoesExist failed: %s", err.Error())
	}
	if !found {
		t.Error("expected shovey to exist")
	}
	found, err = DoesExist(org, "not-a-real-id")
	if err != nil {
		t.Fatalf("DoesExist failed: %s", err.Error())
	}
	if found {
		t.Error("expected missing shovey not to exist")
	}
}

func TestShoveyGet(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	s, err := New(org, "/bin/ls", 300, "100%", []string{"node1"})
	if err != nil {
		t.Fatalf("New failed: %s", err.Error())
	}
	s2, err := Get(org, s.RunID)
	if err != nil {
		t.Fatalf("Get failed: %s", err.Error())
	}
	if s2.Command != "/bin/ls" {
		t.Errorf("expected /bin/ls, got %s", s2.Command)
	}
	if len(s2.NodeNames) != 1 || s2.NodeNames[0] != "node1" {
		t.Errorf("expected [node1], got %v", s2.NodeNames)
	}
}

func TestShoveyAllShoveyIDsAndGetList(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	s1, _ := New(org, "/bin/ls", 300, "100%", []string{})
	s2, _ := New(org, "/bin/cat", 300, "100%", []string{})
	ids, err := AllShoveyIDs(org)
	if err != nil {
		t.Fatalf("AllShoveyIDs failed: %s", err.Error())
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
	list := GetList(org)
	if len(list) != 2 {
		t.Fatalf("expected 2 ids from GetList, got %d", len(list))
	}
	if !((ids[0] == s1.RunID || ids[0] == s2.RunID) && (ids[1] == s1.RunID || ids[1] == s2.RunID)) {
		t.Errorf("returned ids did not match created shoveys")
	}
}

func TestShoveyAllShoveys(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	New(org, "/bin/ls", 300, "100%", []string{})
	New(org, "/bin/cat", 300, "100%", []string{})
	shoveys := AllShoveys(org)
	if len(shoveys) != 2 {
		t.Errorf("expected 2 shoveys, got %d", len(shoveys))
	}
}

func TestShoveyImport(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	shoveyJSON := map[string]interface{}{
		"id":          "imported-run-id",
		"nodes":       []interface{}{"node1", "node2"},
		"command":     "/bin/echo",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
		"status":      "complete",
		"timeout":     float64(120),
		"quorum":      "100%",
	}
	if err := ImportShovey(org, shoveyJSON); err != nil {
		t.Fatalf("ImportShovey failed: %s", err.Error())
	}
	s, err := Get(org, "imported-run-id")
	if err != nil {
		t.Fatalf("Get failed after import: %s", err.Error())
	}
	if s.Command != "/bin/echo" {
		t.Errorf("expected /bin/echo, got %s", s.Command)
	}
}

func TestShoveyImportRun(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	shoveyJSON := map[string]interface{}{
		"id":          "imported-run-id",
		"nodes":       []interface{}{"node1"},
		"command":     "/bin/echo",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
		"status":      "complete",
		"timeout":     float64(120),
		"quorum":      "100%",
	}
	if err := ImportShovey(org, shoveyJSON); err != nil {
		t.Fatalf("ImportShovey failed: %s", err.Error())
	}
	runJSON := map[string]interface{}{
		"run_id":      "imported-run-id",
		"node_name":   "node1",
		"status":      "created",
		"ack_time":    time.Now().Format(time.RFC3339),
		"end_time":    time.Now().Format(time.RFC3339),
		"error":       "",
		"exit_status": float64(0),
	}
	if err := ImportShoveyRun(org, runJSON); err != nil {
		t.Fatalf("ImportShoveyRun failed: %s", err.Error())
	}
	s, _ := Get(org, "imported-run-id")
	run, err := s.GetRun("node1")
	if err != nil {
		t.Fatalf("GetRun failed: %s", err.Error())
	}
	if run.Status != "created" {
		t.Errorf("expected created, got %s", run.Status)
	}
}

func TestShoveyRunUpdateFromJSON(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	shoveyJSON := map[string]interface{}{
		"id":          "imported-run-id",
		"nodes":       []interface{}{"node1"},
		"command":     "/bin/echo",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
		"status":      "running",
		"timeout":     float64(120),
		"quorum":      "100%",
	}
	if err := ImportShovey(org, shoveyJSON); err != nil {
		t.Fatalf("ImportShovey failed: %s", err.Error())
	}
	runJSON := map[string]interface{}{
		"run_id":      "imported-run-id",
		"node_name":   "node1",
		"status":      "created",
		"ack_time":    time.Now().Format(time.RFC3339),
		"end_time":    time.Now().Format(time.RFC3339),
		"error":       "",
		"exit_status": float64(0),
	}
	if err := ImportShoveyRun(org, runJSON); err != nil {
		t.Fatalf("ImportShoveyRun failed: %s", err.Error())
	}
	s, _ := Get(org, "imported-run-id")
	run, _ := s.GetRun("node1")
	update := map[string]interface{}{
		"status":      "succeeded",
		"error":       "none",
		"exit_status": float64(0),
	}
	if err := run.UpdateFromJSON(update); err != nil {
		t.Fatalf("UpdateFromJSON failed: %s", err.Error())
	}
	if run.Status != "succeeded" {
		t.Errorf("expected succeeded, got %s", run.Status)
	}
	if run.Error != "none" {
		t.Errorf("expected none, got %s", run.Error)
	}
}

func TestShoveyAddStreamOutput(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	shoveyJSON := map[string]interface{}{
		"id":          "imported-run-id",
		"nodes":       []interface{}{"node1"},
		"command":     "/bin/echo",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
		"status":      "running",
		"timeout":     float64(120),
		"quorum":      "100%",
	}
	if err := ImportShovey(org, shoveyJSON); err != nil {
		t.Fatalf("ImportShovey failed: %s", err.Error())
	}
	runJSON := map[string]interface{}{
		"run_id":      "imported-run-id",
		"node_name":   "node1",
		"status":      "created",
		"ack_time":    time.Now().Format(time.RFC3339),
		"end_time":    time.Now().Format(time.RFC3339),
		"error":       "",
		"exit_status": float64(0),
	}
	if err := ImportShoveyRun(org, runJSON); err != nil {
		t.Fatalf("ImportShoveyRun failed: %s", err.Error())
	}
	s, _ := Get(org, "imported-run-id")
	run, _ := s.GetRun("node1")
	if err := run.AddStreamOutput("hello", "stdout", 0, true); err != nil {
		t.Fatalf("AddStreamOutput failed: %s", err.Error())
	}
	ds := datastore.New()
	streamKey := fmt.Sprintf("%s_%s_%s_%d", run.ShoveyUUID, run.NodeName, "stdout", 0)
	val, found := ds.Get(org.DataKey("shovey_run_stream"), streamKey)
	if !found {
		t.Error("expected stream output to be saved")
	}
	if val == nil {
		t.Fatal("expected non-nil stream output")
	}
	stream := val.(*ShoveyRunStream)
	if stream.Output != "hello" {
		t.Errorf("expected hello, got %s", stream.Output)
	}
}

func TestShoveyCombineStreamOutput(t *testing.T) {
	org, cleanup := setupShoveyTest(t)
	defer cleanup()

	shoveyJSON := map[string]interface{}{
		"id":          "imported-run-id",
		"nodes":       []interface{}{"node1"},
		"command":     "/bin/echo",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
		"status":      "running",
		"timeout":     float64(120),
		"quorum":      "100%",
	}
	if err := ImportShovey(org, shoveyJSON); err != nil {
		t.Fatalf("ImportShovey failed: %s", err.Error())
	}
	runJSON := map[string]interface{}{
		"run_id":      "imported-run-id",
		"node_name":   "node1",
		"status":      "created",
		"ack_time":    time.Now().Format(time.RFC3339),
		"end_time":    time.Now().Format(time.RFC3339),
		"error":       "",
		"exit_status": float64(0),
	}
	if err := ImportShoveyRun(org, runJSON); err != nil {
		t.Fatalf("ImportShoveyRun failed: %s", err.Error())
	}
	s, _ := Get(org, "imported-run-id")
	run, _ := s.GetRun("node1")
	if err := run.AddStreamOutput("hello ", "stdout", 0, false); err != nil {
		t.Fatalf("AddStreamOutput seq 0 failed: %s", err.Error())
	}
	if err := run.AddStreamOutput("world", "stdout", 1, true); err != nil {
		t.Fatalf("AddStreamOutput seq 1 failed: %s", err.Error())
	}
	combined, err := run.CombineStreamOutput("stdout", 0)
	if err != nil {
		t.Fatalf("CombineStreamOutput failed: %s", err.Error())
	}
	if combined != "hello world" {
		t.Errorf("expected 'hello world', got %s", combined)
	}
}
