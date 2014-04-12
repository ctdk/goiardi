/* User handler functions */

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
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
)

func user_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	path := SplitPath(r.URL.Path)
	user_name := path[1]
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
		case "DELETE":
			chef_user, err := user.Get(user_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			if !opUser.IsAdmin() && !opUser.IsSelf(chef_user) {
				JsonErrorReport(w, r, "Deleting that user is forbidden", http.StatusForbidden)
				return
			}
			/* Docs were incorrect. It does want the body of the
			 * deleted object. */
			json_user := chef_user.ToJson()
			err = chef_user.Delete()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusForbidden)
				return
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&json_user); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		case "GET":
			chef_user, err := user.Get(user_name)

			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			if !opUser.IsAdmin() && !opUser.IsSelf(chef_user) {
				JsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
				return
			}
			
			/* API docs are wrong here re: public_key vs. 
			 * certificate. Also orgname (at least w/ open source)
			 * and clientname, and it wants chef_type and 
			 * json_class
			 */
			json_user := chef_user.ToJson()
			enc := json.NewEncoder(w)
			if err = enc.Encode(&json_user); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		case "PUT":
			user_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			chef_user, err := user.Get(user_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}

			/* Makes chef-pedant happy. I suppose it is, after all,
			 * pedantic. */
			if averr := util.CheckAdminPlusValidator(user_data); averr != nil {
				JsonErrorReport(w, r, averr.Error(), averr.Status())
				return
			}

			if !opUser.IsAdmin() && !opUser.IsSelf(chef_client) {
				JsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
				return
			}
			if !opUser.IsAdmin() {
				var verr util.Gerror
				aerr := opUser.CheckPermEdit(user_data, "admin")
				if aerr != nil {
					JsonErrorReport(w, r, aerr.Error(), aerr.Status())
					return
				}
			}

			json_name, sterr := util.ValidateAsString(user_data["name"])
			if sterr != nil {
				JsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
				return
			}

			/* If user_name and user_data["name"] aren't the
			 * same, we're renaming. Check the new name doesn't
			 * already exist. */
			json_user := chef_user.ToJson()
			if user_name != json_name {
				err := chef_user.Rename(json_name)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else {
					w.WriteHeader(http.StatusCreated)
				}
			} 
			if uerr := chef_user.UpdateFromJson(user_data); uerr != nil {
				JsonErrorReport(w, r, uerr.Error(), uerr.Status())
				return
			}

			if pk, pkfound := user_data["public_key"]; pkfound {
				switch pk := pk.(type){
					case string:
						if pkok, pkerr := user.ValidatePublicKey(pk); !pkok {
							JsonErrorReport(w, r, pkerr.Error(), http.StatusBadRequest)
							return
						}
						chef_user.SetPublicKey(pk)
						json_user["public_key"] = pk
					case nil:
						//show_public_key = false
						;
					default:
						JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
						return
				}
			}

			if p, pfound := user_data["private_key"]; pfound {
				switch p := p.(type) {
					case bool:
						if p {
							if json_user["private_key"], err = chef_user.GenerateKeys(); err != nil {
								JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
								return
							}
							// make sure the json
							// client gets the new
							// public key
							json_user["public_key"] = chef_user.PublicKey()
						}
					default:
						JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
						return
				}
			}

			chef_user.Save()
			
			enc := json.NewEncoder(w)
			if err = enc.Encode(&json_user); err != nil{
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		default:
			JsonErrorReport(w, r, "Unrecognized method for user!", http.StatusMethodNotAllowed)
	}
}
