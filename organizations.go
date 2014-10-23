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
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
)

// might also be best split up
func orgToolHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	pathArray := splitPath(r.URL.Path)
	orgName := vars["org"]

	// Otherwise, it's org work.
	var orgResponse map[string]interface{}

	op := pathArray[2]
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
	switch op {
	case "_validator_key":
		if r.Method == "POST" {
			valname := util.JoinStr(org.Name, "-validator")
			val, err := client.Get(org, valname)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			pem, perr := val.GenerateKeys()
			if perr != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			orgResponse = make(map[string]interface{})
			orgResponse["private_key"] = pem
		} else {
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
	case "association_requests":
		orgResponse = make(map[string]interface{})
		if len(pathArray) == 4 {
			id := vars["id"]
			re := regexp.MustCompile(util.JoinStr("(.+)-", orgName))
			userChk := re.FindStringSubmatch(id)
			if userChk == nil {
				util.JSONErrorReport(w, r, util.JoinStr("Invalid ID ", id, ". Must be of the form username-", orgName) , http.StatusNotFound)
				return
			}
			// Looks like this is supposed to be a delete.
			// TODO: make it do what it should when that bit is in
			orgResponse["id"] = id
			orgResponse["username"] = userChk[1]
		} else {
			switch r.Method {
				case "GET":
					// returns a list of associations with
					// this org. TODO: It should actually
					// do that.
				case "POST":
					// creates the association. TODO: make
					// it do so
					arData, jerr := parseObjJSON(r.Body)
					if jerr != nil {
						jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusCreated)
					orgResponse["uri"] = util.CustomURL(util.JoinStr(r.URL.Path, "/", arData["user"].(string), "-", orgName))
				default:
					jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
					return
			}
		}
	default:
		jsonErrorReport(w, r, "Unknown organization endpoint, rather unlikely to reach", http.StatusBadRequest)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&orgResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func orgMainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	orgName := vars["org"]
	org, err := organization.Get(orgName)
	var orgResponse map[string]interface{}

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

	switch r.Method {
	case "GET", "DELETE":
		orgResponse = org.ToJSON()
		if r.Method == "DELETE" {
			err := org.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		}
	case "PUT":
		jsonErrorReport(w, r, "not implemented", http.StatusNotImplemented)
		return
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&orgResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func orgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var orgResponse map[string]interface{}

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
		validator, pem, err := makeValidator(org)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
		environment.MakeDefaultEnvironment(org)
		orgResponse = org.ToJSON()
		orgResponse["private_key"] = pem
		orgResponse["clientname"] = validator.Name
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&orgResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func makeValidator(org *organization.Organization) (*client.Client, string, util.Gerror) {
	valname := util.JoinStr(org.Name, "-validator")
	val, err := client.New(org, valname)
	if err != nil {
		return nil, "", err
	}
	val.Validator = true
	pem, perr := val.GenerateKeys()
	if perr != nil {
		return nil, "", util.CastErr(perr)
	}
	perr = val.Save()
	if perr != nil {
		return nil, "", util.CastErr(perr)
	}
	return val, pem, nil
}
