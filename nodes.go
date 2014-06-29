/* Node functions */

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
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func nodeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	nodeName := r.URL.Path[7:]

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	/* So, what are we doing? Depends on the HTTP method, of course */
	switch r.Method {
	case "GET", "DELETE":
		if opUser.IsValidator() || !opUser.IsAdmin() && r.Method == "DELETE" && !(opUser.IsClient() && opUser.(*client.Client).NodeName == nodeName) {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		chefNode, err := node.Get(nodeName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(&chefNode); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Method == "DELETE" {
			err = chefNode.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := loginfo.LogEvent(opUser, chefNode, "delete"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
		}
	case "PUT":
		if !opUser.IsAdmin() && !(opUser.IsClient() && opUser.(*client.Client).NodeName == nodeName) {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		nodeData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		chefNode, err := node.Get(nodeName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		/* If nodeName and nodeData["name"] don't match, we
		 * need to make a new node. Make sure that node doesn't
		 * exist. */
		if _, found := nodeData["name"]; !found {
			nodeData["name"] = nodeName
		}
		jsonName, sterr := util.ValidateAsString(nodeData["name"])
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}
		if nodeName != jsonName && jsonName != "" {
			jsonErrorReport(w, r, "Node name mismatch.", http.StatusBadRequest)
			return
		}
		if jsonName == "" {
			nodeData["name"] = nodeName
		}
		nerr := chefNode.UpdateFromJSON(nodeData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return
		}
		err = chefNode.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if lerr := loginfo.LogEvent(opUser, chefNode, "modify"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(&chefNode); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
	}
}
