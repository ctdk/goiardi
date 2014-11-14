/* List handling stuff - a bit general, used by a few handlers */

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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	pathArray := splitPath(r.URL.Path)
	op := pathArray[2]
	var listData map[string]string
	switch op {
	case "roles":
		listData = roleHandling(org, w, r)
	default:
		listData = make(map[string]string)
		listData["huh"] = "not valid"
	}
	if listData != nil {
		enc := json.NewEncoder(w)
		if err := enc.Encode(&listData); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
	}
}

func roleHandling(org *organization.Organization, w http.ResponseWriter, r *http.Request) map[string]string {
	roleResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}
	switch r.Method {
	case "GET":
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		roleList := role.GetList(org)
		for _, k := range roleList {
			itemURL := util.JoinStr("/organizations/", org.Name, "/roles/", k)
			roleResponse[k] = util.CustomURL(itemURL)
		}
	case "POST":
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		roleData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return nil
		}
		if _, ok := roleData["name"].(string); !ok {
			jsonErrorReport(w, r, "Role name missing", http.StatusBadRequest)
			return nil
		}
		chefRole, _ := role.Get(org, roleData["name"].(string))
		if chefRole != nil {
			httperr := fmt.Errorf("Role already exists")
			jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
			return nil
		}
		var nerr util.Gerror
		chefRole, nerr = role.NewFromJSON(org, roleData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return nil
		}
		err := chefRole.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return nil
		}
		if lerr := loginfo.LogEvent(org, opUser, chefRole, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return nil
		}
		roleResponse["uri"] = util.ObjURL(chefRole)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for roles", http.StatusMethodNotAllowed)
		return nil
	}
	return roleResponse
}
