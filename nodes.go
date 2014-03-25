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
	"net/http"
	"encoding/json"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/actor"
)

func node_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	
	node_name := r.URL.Path[7:]

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	/* So, what are we doing? Depends on the HTTP method, of course */
	switch r.Method {
		case "GET", "DELETE":
			if opUser.IsValidator() || !opUser.IsAdmin() && r.Method == "DELETE" && opUser.NodeName != node_name {
				JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			chef_node, err := node.Get(node_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&chef_node); err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if r.Method == "DELETE" {
				err = chef_node.Delete()
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		case "PUT":
			if !opUser.IsAdmin() && opUser.NodeName != node_name {
				JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			node_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			chef_node, err := node.Get(node_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			/* If node_name and node_data["name"] don't match, we
			 * need to make a new node. Make sure that node doesn't
			 * exist. */
			if _, found := node_data["name"]; !found {
				node_data["name"] = node_name
			}
			json_name, sterr := util.ValidateAsString(node_data["name"])
			if sterr != nil {
				JsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
				return
			}
			if node_name != json_name && json_name != "" {
				JsonErrorReport(w, r, "Node name mismatch.", http.StatusBadRequest)
				return
			} else {
				if json_name == "" {
					node_data["name"] = node_name
				}
				nerr := chef_node.UpdateFromJson(node_data)
				if nerr != nil {
					JsonErrorReport(w, r, nerr.Error(), nerr.Status())
					return
				}
			}
			err = chef_node.Save()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&chef_node); err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
		default:
			JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
	}
}
