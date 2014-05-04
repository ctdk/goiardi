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
package log_info

import (
	"testing"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/data_store"
	"time"
)

func TestLogEvent(t *testing.T) {
	doer, _ := client.New("doer")
	obj, _ := client.New("obj")
	err := LogEvent(doer, obj, "create")
	if err != nil {
		t.Errorf(err.Error())
	}
	ds := data_store.New()
	arr := ds.GetLogInfoList()
	if len(arr) != 1 {
		t.Errorf("Too many (or not enough) log events: %d found", len(arr))
	}
	le := arr[0].(*LogInfo)
	if le.Action != "create" {
		t.Errorf("Wrong action")
	}
	if le.Actor != doer {
		t.Errorf("wrong doer")
	}
	if le.ActorType != "client" {
		t.Errorf("wrong actor type, got %s", le.ActorType)
	}
	if le.Object != obj  {
		t.Errorf("wrong object")
	}
	var tdef time.Time
	if le.Time == tdef {
		t.Errorf("no time")
	}
	if le.ExtendedInfo == "" {
		t.Errorf("extended info did not get logged")
	}
}
