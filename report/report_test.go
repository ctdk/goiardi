/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package report

import (
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/pborman/uuid"
	"testing"
	"time"
)

var org *organization.Organization

func init() {
	indexer.Initialize(config.Config)
}

func TestReportCreation(t *testing.T) {
	gob.Register(new(organization.Organization))
	org, _ = organization.New("default", "boo")
	org.Save()
	uuid := "12b8be8d-a2ef-4fc6-88b3-4c18103b88df"
	invalidUUID := "12b8be8d-a2ef-4fc6-88b3-4c18103b88zz"
	r, err := New(org, uuid, "node")
	if err != nil {
		t.Errorf(err.Error())
	}
	if r.RunID != uuid {
		t.Errorf("run ids are not identical: %s :: %s", r.RunID, uuid)
	}
	_, err = New(org, invalidUUID, "node")
	if err == nil {
		t.Errorf("%s created a report, but it shouldn't have", invalidUUID)
	}
	r.Delete()
}

func TestReportUpdating(t *testing.T) {
	create := map[string]interface{}{"action": "start", "run_id": "12b8be8d-a2ef-4fc6-88b3-4c18103b88df", "start_time": "2014-05-10 01:05:42 +0000"}
	//update := map[string]interface{}{"action":"end","resources":[],"status":"success","run_list":[],"total_res_count":"0","data":{},"start_time":"2014-05-10 01:05:42 +0000","end_time":"2014-05-10 01:05:42 +0000"}
	update := map[string]interface{}{"action": "end", "status": "success", "start_time": "2014-05-10 01:05:42 +0000", "end_time": "2014-05-10 01:05:42 +0000", "total_res_count": "0"}
	update["resources"] = make([]interface{}, 0)
	update["run_list"] = "[]"
	update["data"] = make(map[string]interface{})
	r, err := NewFromJSON(org, "node", create)
	if err != nil {
		t.Errorf(err.Error())
	}
	err = r.UpdateFromJSON(update)
	if err != nil {
		t.Errorf(err.Error())
	}
	r.Delete()
}

func TestReportListing(t *testing.T) {
	uuid := "12b8be8d-a2ef-4fc6-88b3-4c18103b88d%d"
	gob.Register(new(Report))
	for i := 0; i < 3; i++ {
		u := fmt.Sprintf(uuid, i)
		r, _ := New(org, u, "node")
		r.StartTime = time.Now()
		r.Save()
	}
	rs := GetList(org)
	if len(rs) != 3 {
		t.Errorf("expected 3 items in list, got %d", len(rs))
	}

	n, _ := node.New(org, "node2")
	for i := 4; i < 6; i++ {
		u := fmt.Sprintf(uuid, i)
		r, _ := New(org, u, n.Name)
		r.StartTime = time.Now()
		r.Save()
	}
	from := time.Now().Add(-(time.Duration(24*90) * time.Hour))
	until := time.Now()
	ns, nerr := GetNodeList(org, n.Name, from, until, 100, "")
	if nerr != nil {
		t.Errorf(nerr.Error())
	}
	if len(ns) != 2 {
		t.Errorf("expected 2 items from node 'node2', got %d", len(ns))
	}

	zs, rerr := GetReportList(org, from, until, 100, "started")
	if rerr != nil {
		t.Errorf(rerr.Error())
	}
	rs = GetList(org)
	if len(zs) != len(rs) {
		t.Errorf("Searching on 'started' status here should have returned everything but it didn't")
	}
	zs, rerr = GetReportList(org, from, until, 100, "success")
	if rerr != nil {
		t.Errorf(rerr.Error())
	}
	if len(zs) != 0 {
		t.Errorf("Searching for successful runs should have returned zero results, but returned %d instead", len(zs))
	}
}

func TestReportCleaning(t *testing.T) {
	// clean out any reports from other tests
	dr := AllReports()
	for _, r := range dr {
		r.Delete()
	}
	n, _ := node.New("deleting_node")
	now := time.Now()

	durations := []time.Duration{1, 2, 5, 2, 4, 10, 7, 14, 15, 19, 20, 26, 100, 320, 24}
	gtTwoWeeks := 8

	from := 14 * 24 * time.Hour
	day := 24 * time.Hour
	for i, d := range durations {
		age := (d * day) + (time.Duration(i) * d * time.Minute)
		st := now.Add(-(age - (5 * time.Minute)))
		et := now.Add(-age)
		u := uuid.New()
		r, _ := New(u, n.Name)
		r.StartTime = st
		r.EndTime = et
		r.Save()
	}
	del, err := DeleteByAge(from)
	if err != nil {
		t.Error(err)
	}
	if del != gtTwoWeeks {
		t.Errorf("%d reports should have been deleted, but reported %d", gtTwoWeeks, del)
	}
	z := AllReports()
	if len(z) != len(durations)-gtTwoWeeks {
		t.Errorf("should have had %d reports left after deleting ones older than two weeks, but had %d", len(durations)-gtTwoWeeks, len(z))
	}
}
