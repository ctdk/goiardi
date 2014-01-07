/* Client functions */

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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/util"
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
				"admin": chef_client.Admin,
				//"orgname": chef_client.Orgname,
				"public_key": chef_client.PublicKey,
			}
			if op != "users" {
				json_client["validator"] = chef_client.Validator
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
				return
			}
			chef_client, err := actor.Get(client_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}

			json_name, sterr := util.ValidateAsString(client_data["name"])
			if sterr != nil {
				JsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
				return
			}

			/* If client_name and client_data["name"] aren't the
			 * same, we're renaming. Check the new name doesn't
			 * already exist. */
			json_client := make(map[string]interface{})
			if client_name != json_name {
				err := chef_client.Rename(json_name)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else {
					w.WriteHeader(http.StatusCreated)
				}
			} 
			if uerr := chef_client.UpdateFromJson(client_data, chef_type); uerr != nil {
				JsonErrorReport(w, r, uerr.Error(), uerr.Status())
				return
			}

			if pk, pkfound := client_data["public_key"]; pkfound {
				switch pk := pk.(type){
					case string:
						// TODO: this needs validation
						chef_client.PublicKey = pk
					case nil:
						//show_public_key = false
						;
					default:
						JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
						return
				}
			}

			if p, pfound := client_data["private_key"]; pfound {
				switch p := p.(type) {
					case bool:
						if p {
							if json_client["private_key"], err = chef_client.GenerateKeys(); err != nil {
								JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
								return
							}
						}
					default:
						JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
						return
				}
			}

			chef_client.Save()
			json_client["name"] = chef_client.Name
			json_client["admin"] = chef_client.Admin
			if op != "users" {
				json_client["validator"] = chef_client.Validator
				json_client["public_key"] = chef_client.PublicKey
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&json_client); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			JsonErrorReport(w, r, "Unrecognized method for client!", http.StatusMethodNotAllowed)
	}
}
