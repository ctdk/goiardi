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
	"encoding/json"
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
)

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	userName := vars["name"]

	var org *organization.Organization
	if orgName, ok := vars["org"]; ok {
		var orgerr util.Gerror
		org, orgerr = organization.Get(orgName)
		if orgerr != nil {
			jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
			return
		}
	}

	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
	case "DELETE":
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "Deleting that user is forbidden", http.StatusForbidden)
			return
		}
		/* Docs were incorrect. It does want the body of the
		 * deleted object. */
		jsonUser := chefUser.ToJSON()

		/* Log the delete event *before* deleting the user, in
		 * case the user is deleting itself. */
		if lerr := loginfo.LogEvent(org, opUser, chefUser, "delete"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		err = chefUser.Delete()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusForbidden)
			return
		}
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonUser); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	case "GET":
		chefUser, err := user.Get(userName)

		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		if !opUser.IsAdmin() && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}

		/* API docs are wrong here re: public_key vs.
		 * certificate. Also orgname (at least w/ open source)
		 * and clientname, and it wants chef_type and
		 * json_class
		 */
		jsonUser := chefUser.ToJSON()
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonUser); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	case "PUT":
		userData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		/* Makes chef-pedant happy. I suppose it is, after all,
		 * pedantic. */
		if averr := util.CheckAdminPlusValidator(userData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return
		}

		if !opUser.IsAdmin() && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}
		if !opUser.IsAdmin() {
			aerr := opUser.CheckPermEdit(userData, "admin")
			if aerr != nil {
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return
			}
		}

		jsonName, sterr := util.ValidateAsString(userData["name"])
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}

		/* If userName and userData["name"] aren't the
		 * same, we're renaming. Check the new name doesn't
		 * already exist. */
		jsonUser := chefUser.ToJSON()
		delete(jsonUser, "public_key")
		if userName != jsonName {
			err := chefUser.Rename(jsonName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			w.WriteHeader(http.StatusCreated)
		}
		if uerr := chefUser.UpdateFromJSON(userData); uerr != nil {
			jsonErrorReport(w, r, uerr.Error(), uerr.Status())
			return
		}

		if pk, pkfound := userData["public_key"]; pkfound {
			switch pk := pk.(type) {
			case string:
				if pkok, pkerr := user.ValidatePublicKey(pk); !pkok {
					jsonErrorReport(w, r, pkerr.Error(), http.StatusBadRequest)
					return
				}
				chefUser.SetPublicKey(pk)
				jsonUser["public_key"] = pk
			case nil:
				//show_public_key = false

			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}

		if p, pfound := userData["private_key"]; pfound {
			switch p := p.(type) {
			case bool:
				if p {
					var perr error
					if jsonUser["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
						jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
						return
					}
					// make sure the json
					// client gets the new
					// public key
					jsonUser["public_key"] = chefUser.PublicKey()
				}
			default:
				jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		}

		serr := chefUser.Save()
		if serr != nil {
			jsonErrorReport(w, r, serr.Error(), serr.Status())
			return
		}
		if lerr := loginfo.LogEvent(org, opUser, chefUser, "modify"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}

		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonUser); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method for user!", http.StatusMethodNotAllowed)
	}
}

func userListHandler(w http.ResponseWriter, r *http.Request) {
	userResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	vars := mux.Vars(r)
	var org *organization.Organization
	if orgName, ok := vars["org"]; ok {
		var orgerr util.Gerror
		org, orgerr = organization.Get(orgName)
		if orgerr != nil {
			jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
			return
		}
	}

	switch r.Method {
	case "GET":
		userList := user.GetList()
		for _, k := range userList {
			/* Make sure it's a client and not a user. */
			itemURL := util.JoinStr("/users/", k)
			userResponse[k] = util.CustomURL(itemURL)
		}
	case "POST":
		userData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		if averr := util.CheckAdminPlusValidator(userData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return
		}
		if !opUser.IsAdmin() && !opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		} else if !opUser.IsAdmin() && opUser.IsValidator() {
			if aerr := opUser.CheckPermEdit(userData, "admin"); aerr != nil {
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return
			}
			if verr := opUser.CheckPermEdit(userData, "validator"); verr != nil {
				jsonErrorReport(w, r, verr.Error(), verr.Status())
				return
			}

		}
		var nameFromJSON interface{}
		var ok bool
		if nameFromJSON, ok = userData["name"]; !ok {
			nameFromJSON, _ = userData["username"]
		}
		userName, sterr := util.ValidateAsString(nameFromJSON)
		if sterr != nil || userName == "" {
			err := fmt.Errorf("Field 'name' missing")
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}

		chefUser, err := user.NewFromJSON(userData)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}

		if publicKey, pkok := userData["public_key"]; !pkok {
			var perr error
			if userResponse["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
				jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			switch publicKey := publicKey.(type) {
			case string:
				if pkok, pkerr := user.ValidatePublicKey(publicKey); !pkok {
					jsonErrorReport(w, r, pkerr.Error(), pkerr.Status())
					return
				}
				chefUser.SetPublicKey(publicKey)
			case nil:

				var perr error
				if userResponse["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
					jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
					return
				}
			default:
				jsonErrorReport(w, r, "Bad public key", http.StatusBadRequest)
				return
			}
		}
		/* If we make it here, we want the public key in the
		 * response. I think. */
		userResponse["public_key"] = chefUser.PublicKey()

		chefUser.Save()
		if lerr := loginfo.LogEvent(org, opUser, chefUser, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		userResponse["uri"] = util.CustomURL(util.JoinStr("/users/", chefUser.Name))
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for clients or users", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&userResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func userAssocHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	userName := vars["name"]

	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	_ = opUser
	if r.Method != "GET" {
		jsonErrorReport(w, r, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, err := user.Get(userName)
	if err != nil {
		jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
		return
	}
	assoc, err := association.GetAllOrgsAssociationReqs(user)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	response := make([]map[string]string, len(assoc))
	for i, a := range assoc {
		ar := make(map[string]string)
		ar["id"] = a.Key()
		ar["orgname"] = a.Org.Name
		response[i] = ar
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func userAssocCountHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "GET" {
		jsonErrorReport(w, r, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	userName := vars["name"]

	logger.Debugf("called count handler")
	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	_ = opUser
	user, err := user.Get(userName)
	if err != nil {
		jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
		return
	}
	count, err := association.OrgsAssociationReqCount(user)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	response := map[string]interface{}{"value": count}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func userAssocIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	userName := vars["name"]

	logger.Debugf("called id handler")

	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	// I think this will be required eventually, but I'm not quite entirely
	// sure how yet
	_ = opUser
	if r.Method != "PUT" {
		jsonErrorReport(w, r, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, err := user.Get(userName)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}

	id := vars["id"]
	re := regexp.MustCompile(util.JoinStr(user.Name, "-(.+)"))
	o := re.FindStringSubmatch(id)
	if o == nil {
		jsonErrorReport(w, r, util.JoinStr("Association request ", id, " is invalid. Must be ", userName, "-orgname."), http.StatusBadRequest)
		return
	}
	org := o[1]
	userData, jerr := parseObjJSON(r.Body)
	if jerr != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
		return
	}
	assoc, err := association.GetReq(id)
	if err != nil {
		jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
		return
	}

	res, ok := userData["response"].(string)
	if !ok {
		jsonErrorReport(w, r, "Param response must be either 'accept' or 'reject'", http.StatusBadRequest)
		return
	}
	switch res {
	case "accept":
		err = assoc.Accept()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
	case "reject":
		err = assoc.Reject()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
	default:
		jsonErrorReport(w, r, "Param response must be either 'accept' or 'reject'", http.StatusBadRequest)
		return
	}
	response := make(map[string]map[string]interface{})
	response["organization"] = map[string]interface{}{"name": org}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
