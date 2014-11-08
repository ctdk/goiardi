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
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/client"
	//"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/organization"
	//"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

// various acl handlers

func orgACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	kind := "containers"
	subkind := "$$root$$"
	a, rerr := acl.Get(org, kind, subkind)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	response := a.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func orgACLEditHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	// always put?
	if r.Method != "PUT" {
		jsonErrorReport(w, r, "unrecognized method", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	kind := "containers"
	subkind := "$$root$$"
	a, rerr := acl.Get(org, kind, subkind)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	perm := vars["perm"]
	aclData, jerr := parseObjJSON(r.Body)
	if jerr != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
		return
	}
	err := a.EditFromJSON(perm, aclData)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	response := a.ToJSON()
	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func containerACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	kind := "containers"
	subkind := vars["name"]
	a, rerr := acl.Get(org, kind, subkind)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	response := a.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func clientACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cl, clerr := client.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	a, rerr := acl.GetItemACL(org, cl)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	response := a.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func groupACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	kind := "groups"
	subkind := vars["group_name"]
	a, rerr := acl.Get(org, kind, subkind)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	response := a.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}