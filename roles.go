/* Role functions */

/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func roleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	/* Roles are bit weird in that there's /roles/NAME, but also
	 * /roles/NAME/environments and /roles/NAME/environments/NAME, so we'll
	 * split up the whole path to get those values. */

	pathArray := splitPath(r.URL.Path)[2:]
	roleName := vars["name"]

	// get HEAD out of the way before the entire role is fetched
	if r.Method == http.MethodHead {
		permCheck := func(r *http.Request, roleName string, opUser actor.Actor) util.Gerror {
			if opUser.IsValidator() {
				return headForbidden()
			}
			return nil
		}
		headChecking(w, r, opUser, roleName, role.DoesExist, permCheck)
		return
	}

	chefRole, err := role.Get(roleName)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
		return
	}

	containerACL, cerr := acl.GetItemACL(org, chefRole)
	if cerr != nil {
		jsonErrorReport(w, r, cerr.Error(), cerr.Status())
		return
	}
	if f, ferr := containerACL.CheckPerm("read", opUser); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}

	if len(pathArray) == 2 {
		/* Normal /roles/NAME case */
		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			delchk, ferr := containerACL.CheckPerm("delete", opUser)
			if ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			}
			if opUser.IsValidator() || (!delchk && r.Method == http.MethodDelete) {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			enc := json.NewEncoder(w)
			if jerr := enc.Encode(&chefRole); jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusInternalServerError)
				return
			}
			if r.Method == http.MethodDelete {
				err = chefRole.Delete()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				err = containerACL.Delete()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, chefRole, "delete"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
			}
		case http.MethodPut:
			if f, ferr := containerACL.CheckPerm("update", opUser); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
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
			if lerr := loginfo.LogEvent(org, opUser, chefRole, "modify"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
			enc := json.NewEncoder(w)
			if jerr = enc.Encode(&chefRole); err != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusInternalServerError)
			}
		default:
			jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		}
	} else {
		var environmentName string
		if len(pathArray) == 4 {
			environmentName = vars["env_name"]
			if _, err := environment.Get(org, environmentName); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
		}
		/* only method for the /roles/NAME/environment stuff is GET */
		switch r.Method {
		case http.MethodGet:
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
				if jerr := enc.Encode(&resp); jerr != nil {
					jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				}
			} else {
				roleEnvs := make([]string, len(chefRole.EnvRunLists)+1)
				roleEnvs[0] = "_default"
				i := 1
				for k := range chefRole.EnvRunLists {
					roleEnvs[i] = k
					i++
				}
				if jerr := enc.Encode(&roleEnvs); jerr != nil {
					jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				}
			}
		default:
			jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		}
	}
}

func roleListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	roleResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	containerACL, cerr := acl.GetContainerACL(org, "roles")
	if cerr != nil {
		jsonErrorReport(w, r, cerr.Error(), cerr.Status())
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
	case "GET":
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		}
		roleList := role.GetList(org)
		for _, k := range roleList {
			itemURL := util.JoinStr("/organizations/", org.Name, "/roles/", k)
			roleResponse[k] = util.CustomURL(itemURL)
		}
	case "POST":
		if f, ferr := containerACL.CheckPerm("create", opUser); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		}
		roleData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		if _, ok := roleData["name"].(string); !ok {
			jsonErrorReport(w, r, "Role name missing", http.StatusBadRequest)
			return
		}
		chefRole, _ := role.Get(org, roleData["name"].(string))
		if chefRole != nil {
			httperr := fmt.Errorf("Role already exists")
			jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
			return
		}
		var nerr util.Gerror
		chefRole, nerr = role.NewFromJSON(org, roleData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return
		}
		err := chefRole.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if lerr := loginfo.LogEvent(org, opUser, chefRole, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		roleResponse["uri"] = util.ObjURL(chefRole)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for roles", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&roleResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
