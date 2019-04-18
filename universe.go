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

package main

import (
	"encoding/json"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/gorilla/mux"
	"net/http"
)

// TODO: Handle orgloader universes

func universeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	switch r.Method {
	case http.MethodGet:
		universe := cookbook.Universe(org)
		enc := json.NewEncoder(w)
		if err := enc.Encode(&universe); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodHead:
		headDefaultResponse(w, r) // Yes, we have a universe.
		return
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
	}
	return
}
