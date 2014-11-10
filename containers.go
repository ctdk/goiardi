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
	"github.com/ctdk/goiardi/container"
	//"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

// container handlers

// may have separate handlers for each kind of container, if warranted.
func containerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	_ = org

	containerName := vars["name"]

	con, cerr := container.Get(org, containerName)
	if cerr != nil {
		jsonErrorReport(w, r, cerr.Error(), cerr.Status())
		return
	}

	response := make(map[string]interface{})
	response["containername"] = con.Name
	response["containerpath"] = con.Name // might be something else
					     // sometimes

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func containerListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	cList := container.GetList(org)
	response := make(map[string]interface{})
	for _, c := range cList {
		conURL := util.JoinStr("/organizations/", org.Name, "/containers/", c)
		response[c] = util.CustomURL(conURL)
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
