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
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
	"log"
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
	if r.Method == "PUT" {
		gData, err := parseObjJSON(r.Body)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Json: %v", gData)
		if gName, ok := gData["groupname"].(string); !ok {
			jsonErrorReport(w, r, "no groupname provided", http.StatusBadRequest)
			return
		} else {
			if gName != groupName {
				errmsg := util.JoinStr("names do not match: ", gName, " ", groupName)
				jsonErrorReport(w, r, errmsg, http.StatusBadRequest)
				return
			}
		}
		log.Printf("groups is: %T", gData["groups"])
		log.Printf("actors is: %T", gData["actors"])
		log.Printf("users is: %T", gData["actors"].(map[string]interface{})["users"])
		if grs, ok := gData["groups"].([]interface{}); ok {
			for _, gn := range grs {
				addGr, err := group.Get(org, gn.(string))
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				err = g.AddGroup(addGr)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			}
		}
		if acts, ok := gData["actors"].(map[string]interface{}); ok {
			if us, uok := acts["users"].([]interface{}); uok {
				for _, un := range us {
					u, err := user.Get(un.(string))
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
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
					c, err := client.Get(org, cn.(string))
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					err = g.AddActor(c)
					if err != nil {
						jsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
				}
			}
		}
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

	groups := group.AllGroups(org)
	response := make(map[string]interface{})
	for _, g := range groups {
		response[g.Name] = util.ObjURL(g)
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
