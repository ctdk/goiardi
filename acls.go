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
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/group"
	//"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/organization"
	//"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

// various acl handlers

func orgACLHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orgName := vars["org"]
	kind := "containers"
	subkind := "$$root$$"
	baseACLPermHandler(w, r, orgName, kind, subkind)
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
	vars := mux.Vars(r)

	orgName := vars["org"]
	kind := "containers"
	subkind := vars["name"]
	baseACLHandler(w, r, orgName, kind, subkind)
}

func clientACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func clientACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cb, clerr := client.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	a, rerr := acl.GetItemACL(org, cb)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}

	aclData, jerr := parseObjJSON(r.Body)
	if jerr != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
		return
	}
	perm := vars["perm"]

	ederr := a.EditFromJSON(perm, aclData)
	if ederr != nil {
		jsonErrorReport(w, r, ederr.Error(), ederr.Status())
		return
	}

	p, ok := a.ACLitems[perm]
	if !ok {
		jsonErrorReport(w, r, "perm nonexistent", http.StatusBadRequest)
		return
	}
	response := p.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func cookbookACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cb, clerr := cookbook.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	a, rerr := acl.GetItemACL(org, cb)
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

func cookbookACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	// Seems to be a PUT only endpoint
	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cb, clerr := cookbook.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}

	aclData, jerr := parseObjJSON(r.Body)
	if jerr != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
		return
	}

	perm := vars["perm"]

	a, rerr := acl.GetItemACL(org, cb)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	ederr := a.EditFromJSON(perm, aclData)
	if ederr != nil {
		jsonErrorReport(w, r, ederr.Error(), ederr.Status())
		return
	}
	p, ok := a.ACLitems[perm]
	if !ok {
		jsonErrorReport(w, r, "perm nonexistent", http.StatusBadRequest)
		return
	}
	response := p.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func groupACLHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	orgName := vars["org"]
	kind := "groups"
	subkind := vars["group_name"]
	baseACLHandler(w, r, orgName, kind, subkind)
}

func groupACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	gb, clerr := group.Get(org, vars["group_name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	perm := vars["perm"]
	
	baseACLPermHandler(w, r, org, gb, perm)
}

func baseACLHandler(w http.ResponseWriter, r *http.Request, orgName string, kind string, subkind string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	org, orgerr := organization.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
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

func baseACLPermHandler(w http.ResponseWriter, r *http.Request, org *organization.Organization, aclOwner acl.ACLOwner, perm string) {
	a, rerr := acl.GetItemACL(org, aclOwner)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}

	aclData, jerr := parseObjJSON(r.Body)
	if jerr != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
		return
	}

	ederr := a.EditFromJSON(perm, aclData)
	if ederr != nil {
		jsonErrorReport(w, r, ederr.Error(), ederr.Status())
		return
	}

	p, ok := a.ACLitems[perm]
	if !ok {
		jsonErrorReport(w, r, "perm nonexistent", http.StatusBadRequest)
		return
	}
	response := p.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
