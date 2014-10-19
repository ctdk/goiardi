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
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func clientHandler(org *organization.Organization, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := splitPath(r.URL.Path)[2:]
	clientName := path[1]
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
	case "DELETE":
		chefClient, gerr := client.Get(org, clientName)
		if gerr != nil {
			jsonErrorReport(w, r, gerr.Error(), gerr.Status())
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefClient) {
			jsonErrorReport(w, r, "Deleting that client is forbidden", http.StatusForbidden)
			return
		}
		/* Docs were incorrect. It does want the body of the
		 * deleted object. */
		jsonClient := chefClient.ToJSON()

		/* Log the delete event before deleting the client, in
		 * case the client is deleting itself. */
		if lerr := loginfo.LogEvent(org, opUser, chefClient, "delete"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		err := chefClient.Delete()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusForbidden)
			return
		}

		enc := json.NewEncoder(w)
		if err = enc.Encode(&jsonClient); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case "GET":
		chefClient, gerr := client.Get(org, clientName)

		if gerr != nil {
			jsonErrorReport(w, r, gerr.Error(), gerr.Status())
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefClient) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}

		/* API docs are wrong here re: public_key vs.
		 * certificate. Also orgname (at least w/ open source)
		 * and clientname, and it wants chef_type and
		 * json_class
		 */
		jsonClient := chefClient.ToJSON()
		enc := json.NewEncoder(w)
		if err := enc.Encode(&jsonClient); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case "PUT":
		clientData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		chefClient, err := client.Get(org, clientName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		/* Makes chef-pedant happy. I suppose it is, after all,
		 * pedantic. */
		if averr := util.CheckAdminPlusValidator(clientData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return
		}

		if !opUser.IsAdmin() && !opUser.IsSelf(chefClient) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}
		if !opUser.IsAdmin() {
			var verr util.Gerror
			aerr := opUser.CheckPermEdit(clientData, "admin")
			if !opUser.IsValidator() {
				verr = opUser.CheckPermEdit(clientData, "validator")
			}
			if aerr != nil && verr != nil {
				jsonErrorReport(w, r, "Client can be either an admin or a validator, but not both.", http.StatusBadRequest)
				return
			} else if aerr != nil || verr != nil {
				if aerr == nil {
					aerr = verr
				}
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return
			}
		}

		jsonName, sterr := util.ValidateAsString(clientData["name"])
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}

		/* If clientName and clientData["name"] aren't the
		 * same, we're renaming. Check the new name doesn't
		 * already exist. */
		jsonClient := chefClient.ToJSON()
		if clientName != jsonName {
			if lerr := loginfo.LogEvent(org, opUser, chefClient, "modify"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
			err := chefClient.Rename(jsonName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			w.WriteHeader(http.StatusCreated)
		}
		if uerr := chefClient.UpdateFromJSON(clientData); uerr != nil {
			jsonErrorReport(w, r, uerr.Error(), uerr.Status())
			return
		}

		if pk, pkfound := clientData["public_key"]; pkfound {
			switch pk := pk.(type) {
			case string:
				if pkok, pkerr := client.ValidatePublicKey(pk); !pkok {
					jsonErrorReport(w, r, pkerr.Error(), http.StatusBadRequest)
					return
				}
				chefClient.SetPublicKey(pk)
				jsonClient["public_key"] = pk
			case nil:
				//show_public_key = false

			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}

		if p, pfound := clientData["private_key"]; pfound {
			switch p := p.(type) {
			case bool:
				if p {
					var cgerr error
					if jsonClient["private_key"], cgerr = chefClient.GenerateKeys(); cgerr != nil {
						jsonErrorReport(w, r, cgerr.Error(), http.StatusInternalServerError)
						return
					}
					// make sure the json
					// client gets the new
					// public key
					jsonClient["public_key"] = chefClient.PublicKey()
				}
			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}
		chefClient.Save()
		if lerr := loginfo.LogEvent(org, opUser, chefClient, "modify"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(&jsonClient); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method for client!", http.StatusMethodNotAllowed)
	}
}
