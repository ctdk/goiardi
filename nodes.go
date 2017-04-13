/* Node functions */

/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	case http.MethodHead:
		permCheck := func(r *http.Request, nodeName string, opUser actor.Actor) util.Gerror {
			if opUser.IsValidator() {
				return headForbidden()
			}
			return nil
		}
		headChecking(w, r, opUser, nodeName, node.DoesExist, permCheck)
		return
	case http.MethodGet, http.MethodDelete:
		if opUser.IsValidator() || !opUser.IsAdmin() && r.Method == http.MethodDelete && !(opUser.IsClient() && opUser.(*client.Client).NodeName == nodeName) {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		chefNode, nerr := node.Get(nodeName)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), http.StatusNotFound)
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(&chefNode); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodDelete {
			err := chefNode.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := loginfo.LogEvent(opUser, chefNode, "delete"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
		}
	case http.MethodPut:
		if !opUser.IsAdmin() && !(opUser.IsClient() && opUser.(*client.Client).NodeName == nodeName) {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		nodeData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		chefNode, kerr := node.Get(nodeName)
		if kerr != nil {
			jsonErrorReport(w, r, kerr.Error(), http.StatusNotFound)
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
		err := chefNode.Save()
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
