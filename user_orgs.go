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

// Seems /users and /organizations/FOO/users are different now, eh.

// user org list handler

import (
	"encoding/json"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
	"log"
)

func userOrgListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	_ = org
	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// so not the right way, exactly, but close enough for now
	userList := user.GetList()
	// I don't even...
	response := make([]map[string]map[string]string, len(userList))
	for i, u := range userList {
		ur := make(map[string]map[string]string)
		ur["user"] = map[string]string{"username": u}
		response[i] = ur
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func userOrgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	userName := vars["name"]

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	var response map[string]interface{}

	switch r.Method {
	case "DELETE":
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		assoc, e := association.GetAssoc(chefUser, org)
		if e != nil {
			log.Printf("Error with org user delete get assoc: %s", e.Error())
		}
		if assoc != nil {
			err = assoc.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		} else {
			key := util.JoinStr(userName, "-", org.Name)
			assocReq, e := association.GetReq(key)
			if e != nil {
				log.Printf("Error with org user delete get req: %s", e.Error())
			}
			if assocReq != nil {
				err = assocReq.Delete()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			} else {
				jsonErrorReport(w, r, "user not in this organization", http.StatusNotFound)
				return
			}
		}
		response = make(map[string]interface{})
		response["response"] = "ok"
	default:
		jsonErrorReport(w, r, "unrecognized method", http.StatusMethodNotAllowed)
		return	
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
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
	if !user.IsSelf(opUser) && !opUser.IsAdmin() {
		jsonErrorReport(w, r, "missing read permission", http.StatusForbidden)
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
	user, err := user.Get(userName)
	if err != nil {
		jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
		return
	}
	if !user.IsSelf(opUser) && !opUser.IsAdmin() {
		jsonErrorReport(w, r, "missing read permission", http.StatusForbidden)
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
