/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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

// Package universe handles the /universe berkshelf API endpoint for goiardi.
package universe

import (
	"encoding/json"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

// UniverseHandler dispatches requests for the /universe endpoint for goiardi.
// It has been moved into a separate module so it can run as a standalone
// Chef Server plugin with a simple wrapper. That's the plan, anyway.
func UniverseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		util.JSONErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	_, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		util.JSONErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if r.Method != "GET" {
		util.JSONErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}

	universe := cookbook.Universe(org)
	enc := json.NewEncoder(w)
	if err := enc.Encode(&universe); err != nil {
		util.JSONErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
