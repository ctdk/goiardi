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
	"fmt"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if !opUser.IsAdmin() {
		jsonErrorReport(w, r, "You must be an admin to do that", http.StatusForbidden)
		return
	}
	pathArray := splitPath(r.URL.Path)

	if len(pathArray) < 3 {
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
		switch pathArray[1] {
		// /status/all/nodes
		case "all":
			if len(pathArray) != 3 {
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
			if pathArray[2] != "nodes" {
				jsonErrorReport(w, r, "Invalid object to get status for", http.StatusBadRequest)
				return
			}
			nodes := node.AllNodes()
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
				nsurl := fmt.Sprintf("/status/node/%s/latest", n.Name)
				sr[i]["url"] = util.CustomURL(nsurl)
			}
			statusResponse = sr
		// /status/node/<nodeName>/(all|latest)
		case "node":
			if len(pathArray) != 4 {
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
			nodeName := pathArray[2]
			op := pathArray[3]
			n, gerr := node.Get(nodeName)
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
