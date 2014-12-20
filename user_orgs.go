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
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
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

	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	containerACL, err := acl.Get(org, "containers", "$$root$$")
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	if f, ferr := containerACL.CheckPerm("read", opUser); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}
	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userList, err := association.UserAssociations(org)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	response := make([]map[string]map[string]string, len(userList))
	for i, u := range userList {
		ur := make(map[string]map[string]string)
		ur["user"] = map[string]string{"username": u.Name}
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

	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	var response map[string]interface{}

	containerACL, err := acl.Get(org, "containers", "$$root$$")
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	if f, ferr := containerACL.CheckPerm("read", opUser); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}

	switch r.Method {
	case "DELETE":
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		if f, ferr := containerACL.CheckPerm("delete", opUser); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f && !opUser.IsSelf(chefUser) {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		} else if f && opUser.IsSelf(chefUser) {
			errMsg := util.JoinStr("Please remove ", chefUser.Name, " from this organization's admins group before removing him or her from the organization.")
			jsonErrorNonArrayReport(w, r, errMsg, http.StatusForbidden)
			return
		}

		assoc, _ := association.GetAssoc(chefUser, org)
		if assoc != nil {
			err = assoc.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			go group.ClearActor(org, chefUser)
			go acl.ResetACLs(org)
		} else {
			key := util.JoinStr(userName, "-", org.Name)
			assocReq, _ := association.GetReq(key)
			if assocReq != nil {
				err = assocReq.Delete()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			} else {
				errMsg := util.JoinStr("Cannot find a user ", userName, " in organization ", org.Name)
				jsonErrorNonArrayReport(w, r, errMsg, http.StatusNotFound)
				return
			}
		}
		response = make(map[string]interface{})
		response["response"] = "ok"
	case "GET":
		chefUser, err := user.Get(userName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		_, err = association.GetAssoc(chefUser, org)
		if err != nil {
			if err.Status() == http.StatusForbidden {
				err = util.Errorf("Cannot find a user %s in organization %s", chefUser.Name, org.Name)
				err.SetStatus(http.StatusNotFound)
			}
			jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
			return
		}
		response = chefUser.ToJSON()
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

	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if r.Method != "PUT" {
		jsonErrorReport(w, r, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, err := user.Get(userName)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}

	if !opUser.IsAdmin() && !opUser.IsSelf(user) {
		jsonErrorReport(w, r, "you may not accept that request on behalf of that user", http.StatusForbidden)
		return
	}

	id := vars["id"]
	re := regexp.MustCompile(util.JoinStr(user.Name, "-(.+)"))
	o := re.FindStringSubmatch(id)
	if o == nil {
		jsonErrorReport(w, r, util.JoinStr("Association request ", id, " is invalid. Must be ", userName, "-orgname."), http.StatusBadRequest)
		return
	}
	org, err := organization.Get(o[1])
	if err != nil {
		jsonErrorNonArrayReport(w, r, err.Error(), err.Status())
		return
	}

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
	// Have to check here if the user who issued the invitation is still an
	// admin for the organization
	containerACL, err := acl.Get(org, "containers", "$$root$$")
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	if f, ferr := containerACL.CheckPerm("update", assoc.Inviter); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorNonArrayReport(w, r, "This invitation is no longer valid. Please notify an administrator and request to be re-invited to the organization.", http.StatusForbidden)
		assoc.Delete()
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
	response["organization"] = map[string]interface{}{"name": org.Name}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
