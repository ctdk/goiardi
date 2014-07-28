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
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/shovey"
	"net/http"
)

func shoveyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if !opUser.IsAdmin() {
		jsonErrorReport(w, r, "you cannot perform this action", http.StatusForbidden)
		return
	}
	pathArray := splitPath(r.URL.Path)
	pathArrayLen := len(pathArray)
	if pathArrayLen < 2 || pathArrayLen > 3 {
		jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	shoveyResponse := make(map[string]interface{})

	switch r.Method {
	case "GET":
		if pathArrayLen == 3 {
			shove, err := shovey.Get(pathArray[2])
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		} else {
			shoveys, err := shovey.AllShoveyJobs()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		}
	case "POST":
		if pathArrayLen == 3 {
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
		}
		shvData, err := parseObjJSON(r.Body)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}

	/*
	n1, _ := node.Get("terqa.local")
	n := []*node.Node{ n1 }
	
	s, err := shovey.New("ls /", 300, "100%", n)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	*/

	enc := json.NewEncoder(w)
	if jerr := enc.Encode(&shoveyResponse); err != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusInternalServerError)
	}

	return
}
