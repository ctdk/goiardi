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

// Log info tests

package loginfo

import (
	"encoding/gob"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"testing"
	"time"
)

func TestLogEvent(t *testing.T) {
	k := make(map[int]interface{})
	gob.Register(k)
	kk := new(LogInfo)
	gob.Register(kk)
	config.Config.LogEvents = true
	doer, _ := client.New("doer")
	obj, _ := client.New("obj")
	gob.Register(doer)
	err := LogEvent(doer, obj, "create")
	if err != nil {
		t.Errorf(err.Error())
	}
	ds := datastore.New()
	arr := ds.GetLogInfoList()
	if len(arr) != 1 {
		t.Errorf("Too many (or not enough) log events: %d found", len(arr))
	}
	arr2, _ := GetLogInfos(nil, 0, 1)
	if len(arr2) != 1 {
		t.Errorf("Something went wrong with variadic args with GetLogInfoList")
	}
	arr3, _ := GetLogInfos(nil)
	if len(arr3) != 1 {
		t.Errorf("Something went wrong with variadic args with no arguments with GetLogInfoList")
	}
	arr4, _ := GetLogInfos(nil, 0)
	if len(arr4) != 1 {
		t.Errorf("Something went wrong with variadic args with one argument with GetLogInfoList")
	}
	le := arr[1].(*LogInfo)
	if le.Action != "create" {
		t.Errorf("Wrong action")
	}
	if le.ActorType != "client" {
		t.Errorf("wrong actor type, got %s", le.ActorType)
	}
	if le.ObjectName != obj.GetName() {
		t.Errorf("wrong object")
	}
	var tdef time.Time
	if le.Time == tdef {
		t.Errorf("no time")
	}
	if le.ExtendedInfo == "" {
		t.Errorf("extended info did not get logged")
	}
	ds.DeleteLogInfo(1)
	arr5 := ds.GetLogInfoList()
	if len(arr5) != 0 {
		t.Errorf("Doesn't look like the logged event got deleted")
	}
	for i := 0; i < 10; i++ {
		LogEvent(doer, obj, "modify")
	}
	arr6 := ds.GetLogInfoList()
	if len(arr6) != 10 {
		t.Errorf("Something went wrong with creating 10 events")
	}
	ds.PurgeLogInfoBefore(5)
	arr7 := ds.GetLogInfoList()
	if len(arr7) != 5 {
		t.Errorf("Should have been 5 events after purging, got %d", len(arr7))
	}
	ds.PurgeLogInfoBefore(10)
	doer2, _ := client.New("doer2")
	for i := 0; i < 10; i++ {
		LogEvent(doer, obj, "modify")
		LogEvent(doer2, obj, "create")
	}
	searchParams := make(map[string]string)
	searchParams["doer"] = doer2.Name
	searching, err := GetLogInfos(searchParams, 0)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(searching) != 10 {
		t.Errorf("len(searching) for log events by doer2 should have returned 10 items, but returned %d instead", len(searching))
	}
}
