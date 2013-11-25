/* Node functions */

/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
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
)

func node_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	
	var node_name string
	if r.Method == "GET" || r.Method == "PUT" || r.Method == "DELETE" {
		node_name = r.URL.Path[7:]
	}

	/* So, what are we doing? Depends on the HTTP method, of course */
	switch r.Method {
		case "GET", "DELETE":
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
			node_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			}
			chef_node, err := node.Get(node_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			/* If node_name and node_data["name"] don't match, we
			 * need to make a new node. Make sure that node doesn't
			 * exist. */
			if node_name != node_data["name"].(string) {
				chef_node, err = node.Get(node_data["name"].(string))
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusConflict)
					return
				} else {
					chef_node, err = node.NewFromJson(node_data)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			} else {
				chef_node.UpdateFromJson(node_data)
			}
			chef_node.Save()
			enc := json.NewEncoder(w)
			if err = enc.Encode(&chef_node); err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
		default:
			JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
	}
}
