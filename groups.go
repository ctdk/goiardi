/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func groupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	groupName := vars["group_name"]
	g, gerr := group.Get(org, groupName)
	if gerr != nil {
		jsonErrorReport(w, r, gerr.Error(), gerr.Status())
		return
	}

	// hmm
	switch r.Method {
	case http.MethodGet:
		if f, err := org.PermCheck.CheckItemPerm(g, opUser, "read"); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "you are not allowed to perform that action", http.StatusForbidden)
			return
		}
	case http.MethodDelete:
		if f, err := org.PermCheck.CheckItemPerm(g, opUser, "delete"); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "you are not allowed to perform that action", http.StatusForbidden)
			return
		}
		err := g.Delete()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
	case http.MethodPut:
		if f, err := org.PermCheck.CheckItemPerm(g, opUser, "update"); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "you are not allowed to perform that action", http.StatusForbidden)
			return
		}
		gData, err := parseObjJSON(r.Body)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		if gName, ok := gData["groupname"].(string); ok {
			if gName != groupName {
				err := g.Rename(gName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				w.WriteHeader(http.StatusCreated)
			}
		}
		statChk := func(s int) int {
			if s == http.StatusNotFound {
				return http.StatusBadRequest
			}
			return s
		}
		ederr := g.Edit(gData["actors"])
		if ederr != nil {
			jsonErrorReport(w, r, ederr.Error(), statChk(ederr.Status()))
			return
		}
		g.Reload()
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}

	response := g.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func groupListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "groups", "read"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}

	var response map[string]interface{}
	switch r.Method {
	case http.MethodGet:
		groups := group.AllGroups(org)
		response = make(map[string]interface{})
		for _, g := range groups {
			response[g.Name] = util.ObjURL(g)
		}
	case http.MethodPost:
		if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "groups", "create"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		gData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}

		var gBase interface{}
		if h, ok := gData["id"]; ok && h != "" {
			gBase = h
		} else if h, ok := gData["groupname"]; ok && h != "" {
			gBase = h
		}
		gName, err := util.ValidateAsString(gBase)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
		}
		if !util.ValidateName(gName) {
			jsonErrorReport(w, r, "invalid group name", http.StatusBadRequest)
			return
		}
		g, err := group.New(org, gName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		err = g.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		// creator perms
		err = org.PermCheck.CreatorOnly(g, opUser)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}

		response = make(map[string]interface{})
		response["uri"] = util.ObjURL(g)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
