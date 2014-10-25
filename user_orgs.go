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

// Seems /users and /organizations/FOO/users are different now, eh.

// user org list handler

import (
	"encoding/json"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/organization"
	//"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func userOrgListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	_ = org
	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// so not the right way, exactly, but close enough for now
	userList := user.GetList()
	// I don't even...
	response := make([]map[string]map[string]string, len(userList))
	for i, u := range userList {
		ur := make(map[string]map[string]string)
		ur["user"] = map[string]string{ "username": u }
		response[i] = ur
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
