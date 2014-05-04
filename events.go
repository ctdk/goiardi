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

// Serve up the list of logged events, and individual events as well.

package main

import (
	"net/http"
	"github.com/ctdk/goiardi/log_info"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/util"
	"encoding/json"
	"strconv"
	"fmt"
)

// The whole list
func event_list_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	
	switch r.Method {
		case "GET":
			if !opUser.IsAdmin() {
				JsonErrorReport(w, r, "You must be an admin to do that", http.StatusForbidden)
				return
			}
			le_list := log_info.GetLogInfos()
			le_resp := make([]map[string]interface{}, len(le_list))
			for i, v := range le_list {
				le_resp[i] = make(map[string]interface{})
				le_resp[i]["event"] = v
				le_url := fmt.Sprintf("/events/%d", v.Id)
				le_resp[i]["url"] = util.CustomURL(le_url)
			}
			enc := json.NewEncoder(w)
			if err := enc.Encode(&le_resp); err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
	}
}

// Individual log events
func event_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	event_id, aerr := strconv.Atoi(r.URL.Path[8:])
	if aerr != nil {
		JsonErrorReport(w, r, aerr.Error(), http.StatusBadRequest)
		return
	}

	switch r.Method {
		case "GET":
			if !opUser.IsAdmin() {
				JsonErrorReport(w, r, "You must be an admin to do that", http.StatusForbidden)
				return
			}
			le, err := log_info.Get(event_id)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&le); err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}

		default:
			JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
	}
}
