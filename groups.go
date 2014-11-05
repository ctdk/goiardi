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
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func groupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	groupName := vars["group_name"]
	log.Printf("group name: %s", groupName)
	log.Printf("Method: %v", r.Method)
	g, gerr := group.Get(org, groupName)
	if gerr != nil {
		jsonErrorReport(w, r, gerr.Error(), gerr.Status())
		return
	}

	// hmm
	switch r.Method {
	case "GET":
		// we're cool
	case "DELETE":
		err := g.Delete()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
	case "PUT":
		gData, err := parseObjJSON(r.Body)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Json: %v", gData)
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
		switch acts := gData["actors"].(type) {
		case map[string]interface{}:
			if us, uok := acts["users"].([]interface{}); uok {
				for _, un := range us {
					unv, err := util.ValidateAsString(un)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					u, err := user.Get(unv)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), statChk(err.Status()))
						return
					}
					err = g.AddActor(u)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
				}
			}
			if cs, cok := acts["clients"].([]interface{}); cok {
				for _, cn := range cs {
					cnv, err := util.ValidateAsString(cn)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					c, err := client.Get(org, cnv)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), statChk(err.Status()))
						return
					}
					err = g.AddActor(c)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
				}
			}
			if grs, ok := acts["groups"].([]interface{}); ok {
				for _, gn := range grs {
					gnv, err := util.ValidateAsString(gn)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					addGr, err := group.Get(org, gnv)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), statChk(err.Status()))
						return
					}
					err = g.AddGroup(addGr)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
				}
			}
			err := g.Save()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		case nil:

		default:
			jsonErrorReport(w, r, "invalid actors for group", http.StatusBadRequest)
			return
		}
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
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	var response map[string]interface{}
	switch r.Method {
	case "GET":
		groups := group.AllGroups(org)
		response = make(map[string]interface{})
		for _, g := range groups {
			response[g.Name] = util.ObjURL(g)
		}
	case "POST":
		gData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("group data: %v", gData)
		//jsonErrorReport(w, r, "Not working yet!", http.StatusNotImplemented)

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
