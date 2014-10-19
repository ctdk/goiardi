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
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func orgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathArray := splitPath(r.URL.Path)
	pathArrayLen := len(pathArray)

	

	// If pathArrayLen is greater than 2, this gets handed off to another
	// handler.
	if pathArrayLen > 2 {
		op := pathArray[2]
		orgName := pathArray[1]

		org, err := organization.Get(orgName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		// check for basic rights to the organization in question,
		// before any beefier checks further down.
		// TODO: do that.

		switch op {
		case "authenticate_user":
			jsonErrorReport(w, r, "so we do use this", http.StatusBadRequest)
		case "clients", "nodes", "roles":
			if pathArrayLen == 3 {
				listHandler(org, w, r)
			} else {
				switch op {
				case "clients":
					clientHandler(org, w, r)
				case "nodes":
					nodeHandler(org, w, r)
				case "roles":
					roleHandler(org, w, r)
				}
			}
		case "cookbooks":
			cookbookHandler(org, w, r)
		case "data":
			dataHandler(org, w, r)
		case "environments":
			environmentHandler(org, w, r)
		case "principals":
			principalHandler(org, w, r)
		case "sandboxes":
			sandboxHandler(org, w, r)
		case "search":
			if pathArray[3] == "reindex" {
				reindexHandler(org, w, r)
			} else {
				searchHandler(org, w, r)
			}
		case "file_store":
			fileStoreHandler(org, w, r)
		case "events":
			if pathArrayLen == 3 {
				eventListHandler(org, w, r)
			} else {
				eventHandler(org, w, r)
			}
		case "reports":
			reportHandler(org, w, r)
		case "universe":
			universeHandler(org, w, r)
		case "shovey":
			shoveyHandler(org, w, r)
		case "status":
			statusHandler(org, w, r)
		case "users":
			// Users may live both under and outside of
			// organizations... Maybe. Docs so far are not
			// very clear. Do this in the meantime.
			if pathArrayLen == 3 {
				orgUserListHandler(org, w, r)
			} else {
				orgUserHandler(org, w, r)
			}
		default:
			jsonErrorReport(w, r, "Unknown endpoint", http.StatusNotFound)
		}
		return

	}
	// Otherwise, it's org work.
	var orgResponse map[string]interface{}

	// Have to do the actor checking and perm stuff a few places here,
	// because we can't assume we'll always have an organization.

	switch pathArrayLen {
	case 2:
		orgName := pathArray[1]

		switch r.Method {
		case "GET":
			org, err := organization.Get(orgName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
			if oerr != nil {
				jsonErrorReport(w, r, oerr.Error(), oerr.Status())
				return
			}
			if !opUser.IsAdmin() {
				jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return
			}
			orgResponse = org.ToJSON()
		case "PUT":
		default:
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
	case 1:
		opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
		if oerr != nil {
			jsonErrorReport(w, r, oerr.Error(), oerr.Status())
			return
		}
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		}
		switch r.Method {
		case "GET":
			orgList := organization.GetList()
			orgResponse = make(map[string]interface{})
			for _, o := range orgList {
				itemURL := fmt.Sprintf("/organizations/%s", o)
				orgResponse[o] = util.CustomURL(itemURL)
			}
		case "POST":
			orgData, jerr := parseObjJSON(r.Body)
			if jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			orgName, verr := util.ValidateAsString(orgData["name"])
			if verr != nil {
				jsonErrorReport(w, r, "field name missing or invalid", http.StatusBadRequest)
				return
			}
			orgFullName, verr := util.ValidateAsString(orgData["full_name"])
			if verr != nil {
				jsonErrorReport(w, r, "field full name missing or invalid", http.StatusBadRequest)
				return
			}
			org, err := organization.New(orgName, orgFullName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			environment.MakeDefaultEnvironment(org)
			orgResponse = org.ToJSON()
		default:
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&orgResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
