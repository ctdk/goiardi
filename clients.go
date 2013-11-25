/* Client functions */

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
	"github.com/ctdk/goiardi/actor"
	"fmt"
	"strings"
)

func actor_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	path := SplitPath(r.URL.Path)
	op := path[0]
	client_name := path[1]
	chef_type := strings.TrimSuffix(op, "s")
	/* Make sure we aren't trying anything with a user */
	c_chk, _ := actor.Get(client_name)
	if (c_chk != nil && c_chk.ChefType != chef_type){
		var other_actor_type string
		if chef_type == "client" {
			other_actor_type = "user"
		} else {
			other_actor_type = "client"
		}
		err_user := fmt.Errorf("%s is a %s, not a client.", client_name, chef_type, other_actor_type)
		JsonErrorReport(w, r, err_user.Error(), http.StatusBadRequest)
		return
	}
	switch r.Method {
		case "DELETE":
			/* no response body here */
			chef_client, err := actor.Get(client_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			err = chef_client.Delete()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			/* Otherwise, we don't actually do anything. */
		case "GET":
			chef_client, err := actor.Get(client_name)

			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			/* API docs are wrong here re: public_key vs. 
			 * certificate. Also orgname (at least w/ open source)
			 * and clientname, and it wants chef_type and 
			 * json_class
			 */
			json_client := map[string]interface{}{
				"name": chef_client.Name, // set same as above
							  // for now
				"chef_type": chef_client.ChefType,
				"json_class": chef_client.JsonClass,
				"validator": chef_client.Validator,
				"admin": chef_client.Admin,
				//"orgname": chef_client.Orgname,
				"public_key": chef_client.PublicKey,
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&json_client); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		case "PUT":
			client_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			}
			chef_client, err := actor.Get(client_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}

			if _, nok := client_data["name"].(string); !nok {
				JsonErrorReport(w, r, "Client name does not appear to be a string", http.StatusBadRequest)
				return
			}

			/* If client_name and client_data["name"] aren't the
			 * same, we're renaming. Check the new name doesn't
			 * already exist. */
			json_client := make(map[string]interface{})
			if client_name != client_data["name"].(string) {
				rename_client, _ := actor.Get(client_data["name"].(string))
				if rename_client != nil {
					conflict_err := fmt.Errorf("Client (or user) %s already exists, can't rename.", client_data["name"].(string))
					JsonErrorReport(w, r, conflict_err.Error(), http.StatusConflict)
					return
				} else {
					if err = chef_client.Rename(client_data["name"].(string)); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					if json_client["private_key"], err = chef_client.GenerateKeys(); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusCreated)
				}
			} else {
				if client_data["private_key"] != nil {
					if json_client["private_key"], err = chef_client.GenerateKeys(); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}

			if t, ok := client_data["admin"].(bool); ok {
				chef_client.Admin = t
			}
			if v, vok := client_data["validator"].(bool); vok {
				chef_client.Validator = v
			}
			chef_client.Save()
			json_client["name"] = chef_client.Name
			json_client["admin"] = chef_client.Admin
			json_client["validator"] = chef_client.Validator
			enc := json.NewEncoder(w)
			if err = enc.Encode(&json_client); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			JsonErrorReport(w, r, "Unrecognized method for client!", http.StatusMethodNotAllowed)
	}
}
