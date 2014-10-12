/* Role functions */

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
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func roleHandler(org *organization.Organization, w http.ResponseWriter, r *http.Request) {
	_ = org
	w.Header().Set("Content-Type", "application/json")

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	/* Roles are bit weird in that there's /roles/NAME, but also
	 * /roles/NAME/environments and /roles/NAME/environments/NAME, so we'll
	 * split up the whole path to get those values. */

	pathArray := splitPath(r.URL.Path)[2:]
	roleName := pathArray[1]

	chefRole, err := role.Get(roleName)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
		return
	}

	if len(pathArray) == 2 {
		/* Normal /roles/NAME case */
		switch r.Method {
		case "GET", "DELETE":
			if opUser.IsValidator() || (!opUser.IsAdmin() && r.Method == "DELETE") {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&chefRole); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if r.Method == "DELETE" {
				err = chefRole.Delete()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				if lerr := loginfo.LogEvent(opUser, chefRole, "delete"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
			}
		case "PUT":
			if !opUser.IsAdmin() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			roleData, jerr := parseObjJSON(r.Body)
			if jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			if _, ok := roleData["name"]; !ok {
				roleData["name"] = roleName
			}
			jsonName, sterr := util.ValidateAsString(roleData["name"])
			if sterr != nil {
				jsonErrorReport(w, r, sterr.Error(), sterr.Status())
				return
			}
			if roleName != roleData["name"].(string) {
				jsonErrorReport(w, r, "Role name mismatch", http.StatusBadRequest)
				return
			}
			if jsonName == "" {
				roleData["name"] = roleName
			}
			nerr := chefRole.UpdateFromJSON(roleData)
			if nerr != nil {
				jsonErrorReport(w, r, nerr.Error(), nerr.Status())
				return
			}

			err = chefRole.Save()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := loginfo.LogEvent(opUser, chefRole, "modify"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
			enc := json.NewEncoder(w)
			if err = enc.Encode(&chefRole); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
		default:
			jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		}
	} else {
		var environmentName string
		if len(pathArray) == 4 {
			environmentName = pathArray[3]
			if _, err := environment.Get(environmentName); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
		}
		/* only method for the /roles/NAME/environment stuff is GET */
		switch r.Method {
		case "GET":
			/* If we have an environment name, return the
			 * environment specific run_list. Otherwise,
			 * return the environments we have run lists
			 * for. Always at least return "_default",
			 * which refers to run_list. */
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}

			enc := json.NewEncoder(w)
			if environmentName != "" {
				var runList []string
				if environmentName == "_default" {
					runList = chefRole.RunList
				} else {
					runList = chefRole.EnvRunLists[environmentName]
				}
				resp := make(map[string][]string, 1)
				resp["run_list"] = runList
				if err = enc.Encode(&resp); err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				}
			} else {
				roleEnvs := make([]string, len(chefRole.EnvRunLists)+1)
				roleEnvs[0] = "_default"
				i := 1
				for k := range chefRole.EnvRunLists {
					roleEnvs[i] = k
					i++
				}
				if err = enc.Encode(&roleEnvs); err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				}
			}
		default:
			jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		}
	}
}
