/* Principals functions */

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
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func principalHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	errMap := make(map[string]interface{})

	if orgerr != nil {
		if orgerr.Status() == http.StatusNotFound {
			errMap["not_found"] = "org"
			errMap["error"] = util.JoinStr("Cannot find org ", vars["org"])
			util.JSONErrorMapReport(w, r, errMap, http.StatusNotFound)
			return
		}
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	principalName := vars["name"]
	if principalName == "" {
		jsonErrorReport(w, r, "no principal name given", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		chefActor, err := actor.GetReqUser(org, principalName)
		if err != nil {
			errMsg := util.JoinStr("Cannot find principal ", principalName)
			errMap["not_found"] = "principal"
			errMap["error"] = errMsg
			util.JSONErrorMapReport(w, r, errMap, http.StatusNotFound)
			return
		}
		var chefType string
		var orgMember bool
		if chefActor.IsUser() {
			chefType = "user"
			ac, _ := association.GetAssoc(chefActor.(*user.User), org)
			if ac != nil {
				orgMember = true
			}
		} else {
			chefType = "client"
			orgMember = true
		}
		jsonPrincipal := map[string]interface{}{
			"name":       chefActor.GetName(),
			"type":       chefType,
			"public_key": chefActor.PublicKey(),
			"org_member": orgMember,
			"authz_id": chefActor.Authz(),
		}
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonPrincipal); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method for principals!", http.StatusMethodNotAllowed)
	}
}
