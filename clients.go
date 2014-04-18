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
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/util"
	"log"
)

func client_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	path := SplitPath(r.URL.Path)
	client_name := path[1]
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
		case "DELETE":
			chef_client, gerr := client.Get(client_name)
			if gerr != nil {
				JsonErrorReport(w, r, gerr.Error(), gerr.Status())
				return
			}
			if !opUser.IsAdmin() && !opUser.IsSelf(chef_client) {
				JsonErrorReport(w, r, "Deleting that client is forbidden", http.StatusForbidden)
				return
			}
			/* Docs were incorrect. It does want the body of the
			 * deleted object. */
			json_client := chef_client.ToJson()
			err := chef_client.Delete()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusForbidden)
				return
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&json_client); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		case "GET":
			chef_client, gerr := client.Get(client_name)

			if gerr != nil {
				JsonErrorReport(w, r, gerr.Error(), gerr.Status())
				return
			}
			if !opUser.IsAdmin() && !opUser.IsSelf(chef_client) {
				JsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
				return
			}
			
			/* API docs are wrong here re: public_key vs. 
			 * certificate. Also orgname (at least w/ open source)
			 * and clientname, and it wants chef_type and 
			 * json_class
			 */
			json_client := chef_client.ToJson()
			enc := json.NewEncoder(w)
			if err := enc.Encode(&json_client); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		case "PUT":
			client_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			chef_client, err := client.Get(client_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}

			/* Makes chef-pedant happy. I suppose it is, after all,
			 * pedantic. */
			if averr := util.CheckAdminPlusValidator(client_data); averr != nil {
				JsonErrorReport(w, r, averr.Error(), averr.Status())
				return
			}

			if !opUser.IsAdmin() && !opUser.IsSelf(chef_client) {
				JsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
				return
			}
			if !opUser.IsAdmin() {
				var verr util.Gerror
				aerr := opUser.CheckPermEdit(client_data, "admin")
				if !opUser.IsValidator() {
					verr = opUser.CheckPermEdit(client_data, "validator")
				}
				if aerr != nil && verr != nil {
					JsonErrorReport(w, r, "Client can be either an admin or a validator, but not both.", http.StatusBadRequest)
					return
				} else if aerr != nil || verr != nil {
					if aerr == nil {
						aerr = verr
					}
					JsonErrorReport(w, r, aerr.Error(), aerr.Status())
					return
				}
			}

			json_name, sterr := util.ValidateAsString(client_data["name"])
			if sterr != nil {
				JsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
				return
			}

			/* If client_name and client_data["name"] aren't the
			 * same, we're renaming. Check the new name doesn't
			 * already exist. */
			json_client := chef_client.ToJson()
			if client_name != json_name {
				err := chef_client.Rename(json_name)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else {
					w.WriteHeader(http.StatusCreated)
				}
			} 
			if uerr := chef_client.UpdateFromJson(client_data); uerr != nil {
				JsonErrorReport(w, r, uerr.Error(), uerr.Status())
				return
			}

			if pk, pkfound := client_data["public_key"]; pkfound {
				switch pk := pk.(type){
					case string:
						if pkok, pkerr := client.ValidatePublicKey(pk); !pkok {
							JsonErrorReport(w, r, pkerr.Error(), http.StatusBadRequest)
							return
						}
						chef_client.SetPublicKey(pk)
						json_client["public_key"] = pk
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
							var cgerr error
							if json_client["private_key"], cgerr = chef_client.GenerateKeys(); cgerr != nil {
								JsonErrorReport(w, r, cgerr.Error(), http.StatusInternalServerError)
								return
							}
							// make sure the json
							// client gets the new
							// public key
							json_client["public_key"] = chef_client.PublicKey()
						}
					default:
						JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
						return
				}
			}
			chef_client.Save()
			
			enc := json.NewEncoder(w)
			if err := enc.Encode(&json_client); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			JsonErrorReport(w, r, "Unrecognized method for client!", http.StatusMethodNotAllowed)
	}
}
