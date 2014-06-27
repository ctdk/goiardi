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

package report

import (
	"testing"
	"fmt"
	"github.com/ctdk/goiardi/node"
	"time"
	"encoding/gob"
)

func TestReportCreation(t *testing.T){
	uuid := "12b8be8d-a2ef-4fc6-88b3-4c18103b88df"
	invalid_uid := "12b8be8d-a2ef-4fc6-88b3-4c18103b88zz"
	r, err := New(uuid, "node")
	if err != nil {
		t.Errorf(err.Error())
	}
	if r.RunId != uuid {
		t.Errorf("run ids are not identical: %s :: %s", r.RunId, uuid)
	}
	_, err = New(invalid_uid, "node")
	if err == nil {
		t.Errorf("%s created a report, but it shouldn't have")
	}
	r.Delete()
}

func TestReportUpdating(t *testing.T){
	create := map[string]interface{}{"action":"start","run_id":"12b8be8d-a2ef-4fc6-88b3-4c18103b88df","start_time":"2014-05-10 01:05:42 +0000"}
	//update := map[string]interface{}{"action":"end","resources":[],"status":"success","run_list":[],"total_res_count":"0","data":{},"start_time":"2014-05-10 01:05:42 +0000","end_time":"2014-05-10 01:05:42 +0000"}
	update := map[string]interface{}{"action":"end", "status":"success", "start_time":"2014-05-10 01:05:42 +0000","end_time":"2014-05-10 01:05:42 +0000", "total_res_count":"0"  }
	update["resources"] = make([]interface{},0)
	update["run_list"] = "[]"
	update["data"] = make(map[string]interface{})
	r, err := NewFromJson("node", create)
	if err != nil {
		t.Errorf(err.Error())
	}
	err = r.UpdateFromJson(update)
	if err != nil {
		t.Errorf(err.Error())
	}
	r.Delete()
}

func TestReportListing(t *testing.T){
	uuid := "12b8be8d-a2ef-4fc6-88b3-4c18103b88d%d"
	gob.Register(new(Report))
	for i := 0; i < 3; i++ {
		u := fmt.Sprintf(uuid, i)
		r, _ := New(u, "node")
		r.StartTime = time.Now()
		r.Save()
	}
	rs := GetList()
	if len(rs) != 3 {
		t.Errorf("expected 3 items in list, got %d", len(rs))
	}

	n, _ := node.New("node2")
	for i := 4; i < 6; i++ {
		u := fmt.Sprintf(uuid, i)
		r, _ := New(u, n.Name)
		r.StartTime = time.Now()
		r.Save()
	}
	from := time.Now().Add(-(time.Duration(24 * 90) * time.Hour))
	until := time.Now()
	ns, nerr := GetNodeList(n.Name, from, until, 100, "")
	if nerr != nil {
		t.Errorf(nerr.Error())
	}
	if len(ns) != 2 {
		t.Errorf("expected 2 items from node 'node2', got %d", len(ns))
	}

	zs, rerr := GetReportList(from, until, 100, "started")
	if rerr != nil {
		t.Errorf(rerr.Error())
	}
	rs = GetList()
	if len(zs) != len(rs) {
		t.Errorf("Searching on 'started' status here should have returned everything but it didn't")
	}
	zs, rerr = GetReportList(from, until, 100, "success")
	if rerr != nil {
		t.Errorf(rerr.Error())
	}
	if len(zs) != 0 {
		t.Errorf("Searching for successful runs should have returned zero results, but returned %d instead", len(zs))
	}
}
