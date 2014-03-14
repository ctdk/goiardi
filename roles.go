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
	"net/http"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/environment"
	"encoding/json"
	"github.com/ctdk/goiardi/actor"
)

func role_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	/* Roles are bit weird in that there's /roles/NAME, but also
	 * /roles/NAME/environments and /roles/NAME/environments/NAME, so we'll
	 * split up the whole path to get those values. */

	path_array := SplitPath(r.URL.Path)
	role_name := path_array[1]

	chef_role, err := role.Get(role_name)
	if err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
		return
	}

	if len(path_array) == 2 {
		/* Normal /roles/NAME case */
		switch r.Method {
			case "GET", "DELETE":
				if opUser.IsValidator() || (!opUser.IsAdmin() && r.Method == "DELETE") {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				enc := json.NewEncoder(w)
				if err = enc.Encode(&chef_role); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				if r.Method == "DELETE" {
					err = chef_role.Delete()
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			case "PUT":
				if !opUser.IsAdmin() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				role_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
					return
				}
				if _, ok := role_data["name"]; !ok {
					role_data["name"] = role_name
				}
				json_name, sterr := util.ValidateAsString(role_data["name"])
				if sterr != nil {
					JsonErrorReport(w, r, sterr.Error(), sterr.Status())
					return
				}
				if role_name != role_data["name"].(string) {
					JsonErrorReport(w, r, "Role name mismatch", http.StatusBadRequest)
					return
				} else {
					if json_name == "" {
						role_data["name"] = role_name
					}
					nerr := chef_role.UpdateFromJson(role_data)
					if nerr != nil {
						JsonErrorReport(w, r, nerr.Error(), nerr.Status())
						return
					}
				}
	
				chef_role.Save()
				enc := json.NewEncoder(w)
				if err = enc.Encode(&chef_role); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				}
			default:
				JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		}
	} else {
		var environment_name string
		if len(path_array) == 4{
			environment_name = path_array[3]
			if _, err := environment.Get(environment_name); err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
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
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}

				enc := json.NewEncoder(w)
				if environment_name != "" {
					var run_list []string
					if environment_name == "_default" {
						run_list = chef_role.RunList
					} else {
						run_list = chef_role.EnvRunLists[environment_name]
					}
					resp := make(map[string][]string, 1)
					resp["run_list"] = run_list
					if err = enc.Encode(&resp); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					}
				} else {
					role_envs := make([]string, len(chef_role.EnvRunLists) + 1)
					role_envs[0] = "_default"
					i := 1
					for k, _ := range chef_role.EnvRunLists {
						role_envs[i] = k
						i++
					}
					if err = enc.Encode(&role_envs); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					}
				}
			default:
				JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		}
	}
}
