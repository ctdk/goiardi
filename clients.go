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
	"encoding/json"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/log_info"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func clientHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := SplitPath(r.URL.Path)
	clientName := path[1]
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JSONErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
	case "DELETE":
		chefClient, gerr := client.Get(clientName)
		if gerr != nil {
			JSONErrorReport(w, r, gerr.Error(), gerr.Status())
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefClient) {
			JSONErrorReport(w, r, "Deleting that client is forbidden", http.StatusForbidden)
			return
		}
		/* Docs were incorrect. It does want the body of the
		 * deleted object. */
		jsonClient := chefClient.ToJson()

		/* Log the delete event before deleting the client, in
		 * case the client is deleting itself. */
		if lerr := log_info.LogEvent(opUser, chefClient, "delete"); lerr != nil {
			JSONErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		err := chefClient.Delete()
		if err != nil {
			JSONErrorReport(w, r, err.Error(), http.StatusForbidden)
			return
		}

		enc := json.NewEncoder(w)
		if err = enc.Encode(&jsonClient); err != nil {
			JSONErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case "GET":
		chefClient, gerr := client.Get(clientName)

		if gerr != nil {
			JSONErrorReport(w, r, gerr.Error(), gerr.Status())
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefClient) {
			JSONErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}

		/* API docs are wrong here re: public_key vs.
		 * certificate. Also orgname (at least w/ open source)
		 * and clientname, and it wants chef_type and
		 * json_class
		 */
		jsonClient := chefClient.ToJson()
		enc := json.NewEncoder(w)
		if err := enc.Encode(&jsonClient); err != nil {
			JSONErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case "PUT":
		clientData, jerr := ParseObjJson(r.Body)
		if jerr != nil {
			JSONErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		chefClient, err := client.Get(clientName)
		if err != nil {
			JSONErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		/* Makes chef-pedant happy. I suppose it is, after all,
		 * pedantic. */
		if averr := util.CheckAdminPlusValidator(clientData); averr != nil {
			JSONErrorReport(w, r, averr.Error(), averr.Status())
			return
		}

		if !opUser.IsAdmin() && !opUser.IsSelf(chefClient) {
			JSONErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}
		if !opUser.IsAdmin() {
			var verr util.Gerror
			aerr := opUser.CheckPermEdit(clientData, "admin")
			if !opUser.IsValidator() {
				verr = opUser.CheckPermEdit(clientData, "validator")
			}
			if aerr != nil && verr != nil {
				JSONErrorReport(w, r, "Client can be either an admin or a validator, but not both.", http.StatusBadRequest)
				return
			} else if aerr != nil || verr != nil {
				if aerr == nil {
					aerr = verr
				}
				JSONErrorReport(w, r, aerr.Error(), aerr.Status())
				return
			}
		}

		jsonName, sterr := util.ValidateAsString(clientData["name"])
		if sterr != nil {
			JSONErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}

		/* If clientName and clientData["name"] aren't the
		 * same, we're renaming. Check the new name doesn't
		 * already exist. */
		jsonClient := chefClient.ToJson()
		if clientName != jsonName {
			if lerr := log_info.LogEvent(opUser, chefClient, "modify"); lerr != nil {
				JSONErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
			err := chefClient.Rename(jsonName)
			if err != nil {
				JSONErrorReport(w, r, err.Error(), err.Status())
				return
			}
			w.WriteHeader(http.StatusCreated)
		}
		if uerr := chefClient.UpdateFromJson(clientData); uerr != nil {
			JSONErrorReport(w, r, uerr.Error(), uerr.Status())
			return
		}

		if pk, pkfound := clientData["public_key"]; pkfound {
			switch pk := pk.(type) {
			case string:
				if pkok, pkerr := client.ValidatePublicKey(pk); !pkok {
					JSONErrorReport(w, r, pkerr.Error(), http.StatusBadRequest)
					return
				}
				chefClient.SetPublicKey(pk)
				jsonClient["public_key"] = pk
			case nil:
				//show_public_key = false

			default:
				JSONErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}

		if p, pfound := clientData["private_key"]; pfound {
			switch p := p.(type) {
			case bool:
				if p {
					var cgerr error
					if jsonClient["private_key"], cgerr = chefClient.GenerateKeys(); cgerr != nil {
						JSONErrorReport(w, r, cgerr.Error(), http.StatusInternalServerError)
						return
					}
					// make sure the json
					// client gets the new
					// public key
					jsonClient["public_key"] = chefClient.PublicKey()
				}
			default:
				JSONErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}
		chefClient.Save()
		if lerr := log_info.LogEvent(opUser, chefClient, "modify"); lerr != nil {
			JSONErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(&jsonClient); err != nil {
			JSONErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		JSONErrorReport(w, r, "Unrecognized method for client!", http.StatusMethodNotAllowed)
	}
}
