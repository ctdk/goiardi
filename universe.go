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

package main

import (
	"encoding/json"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/organization"
	"net/http"
)

func universeHandler(org *organization.Organization, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if r.Method != "GET" {
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}

	universe := cookbook.Universe(org)
	enc := json.NewEncoder(w)
	if err := enc.Encode(&universe); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
