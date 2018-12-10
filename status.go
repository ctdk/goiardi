/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "nodes", "update"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}
	pathArray := splitPath(r.URL.Path)[2:]
	pathArrayLen := len(pathArray)

	if pathArrayLen < 3 {
		jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	var statusResponse interface{}

	switch r.Method {
	case http.MethodHead:
		// HEAD responses don't look real meaningful here
		headDefaultResponse(w, r)
		return
	case http.MethodGet:
		/* pathArray[1] will tell us what operation we're doing */
		switch vars["specif"] {
		// /status/all/nodes
		case "all":
			if pathArrayLen != 3 {
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
			if pathArray[2] != "nodes" {
				jsonErrorReport(w, r, "Invalid object to get status for", http.StatusBadRequest)
				return
			}
			nodes := node.AllNodes(org)
			sr := make([]map[string]string, len(nodes))
			for i, n := range nodes {
				ns, err := n.LatestStatus()
				if err != nil {
					nsbad := make(map[string]string)
					nsbad["node_name"] = n.Name
					nsbad["status"] = "no record"
					sr[i] = nsbad
					continue
				}
				sr[i] = ns.ToJSON()
				nsurl := util.JoinStr("/organizations/", org.Name, "/status/node/", n.Name, "/latest")
				sr[i]["url"] = util.CustomURL(nsurl)
			}
			statusResponse = sr
		// /status/node/<nodeName>/(all|latest)
		case "node":
			if pathArrayLen != 4 {
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
			nodeName := vars["node_name"]
			op := vars["op"]
			n, gerr := node.Get(org, nodeName)
			if gerr != nil {
				jsonErrorReport(w, r, gerr.Error(), gerr.Status())
				return
			}
			switch op {
			case "latest":
				ns, err := n.LatestStatus()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				statusResponse = ns.ToJSON()
			case "all":
				ns, err := n.AllStatuses()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				sr := make([]map[string]string, len(ns))
				for i, v := range ns {
					sr[i] = v.ToJSON()
				}
				statusResponse = sr
			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		default:
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
		}
	default:
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&statusResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
