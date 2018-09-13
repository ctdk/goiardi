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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/container"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

// various acl handlers

func orgACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	orgName := vars["org"]
	kind := "containers"
	subkind := "$$root$$"
	baseACLHandler(w, r, orgName, kind, subkind)
}

type responder interface {
	ToJSON() map[string]interface{}
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
	org, orgerr := orgloader.Get(orgName)
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
	if aerr := aclPermCheck(r, org, a, "grant"); aerr != nil {
		jsonErrorReport(w, r, aerr.Error(), aerr.Status())
		return
	}
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

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := container.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	baseItemACLHandler(w, r, org, it)
}

func containerACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := container.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, it, perm)
}

func clientACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cl, clerr := client.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	baseItemACLHandler(w, r, org, cl)
}

func clientACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cl, clerr := client.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, cl, perm)
}

func cookbookACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cb, clerr := cookbook.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	baseItemACLHandler(w, r, org, cb)
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
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	cb, clerr := cookbook.Get(org, vars["name"])
	if clerr != nil {
		jsonErrorReport(w, r, clerr.Error(), clerr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, cb, perm)
}

func groupACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := group.Get(org, vars["group_name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	baseItemACLHandler(w, r, org, it)

}

func groupACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
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

func environmentACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := environment.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	baseItemACLHandler(w, r, org, it)
}

func environmentACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := environment.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, it, perm)
}

func nodeACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := node.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	baseItemACLHandler(w, r, org, it)
}

func nodeACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := node.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, it, perm)
}

func roleACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := role.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	baseItemACLHandler(w, r, org, it)
}

func roleACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := role.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, it, perm)
}

func dataACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := databag.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	baseItemACLHandler(w, r, org, it)
}

func dataACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := databag.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, it, perm)
}

func baseACLHandler(w http.ResponseWriter, r *http.Request, orgName string, kind string, subkind string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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
	if pok, perr := org.PermCheck(opUser, "grant"); perr != nil {
		jsonErrorReport(w, r, perr.Error(), perr.Status())
		return
	} else if !pok {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}

	a, rerr := acl.Get(org, kind, subkind)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	if aerr := aclPermCheck(r, org, a, "grant"); aerr != nil {
		jsonErrorReport(w, r, aerr.Error(), aerr.Status())
		return
	}
	response := a.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func baseItemACLHandler(w http.ResponseWriter, r *http.Request, org *organization.Organization, aclOwner acl.ACLOwner) {
	a, rerr := acl.GetItemACL(org, aclOwner)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	if aerr := aclPermCheck(r, org, a, "grant"); aerr != nil {
		jsonErrorReport(w, r, aerr.Error(), aerr.Status())
		return
	}
	sendResponse(w, r, a)
}

func baseACLPermHandler(w http.ResponseWriter, r *http.Request, org *organization.Organization, aclOwner acl.ACLOwner, perm string) {
	a, rerr := acl.GetItemACL(org, aclOwner)
	if rerr != nil {
		jsonErrorReport(w, r, rerr.Error(), rerr.Status())
		return
	}
	if aerr := aclPermCheck(r, org, a, "grant"); aerr != nil {
		jsonErrorReport(w, r, aerr.Error(), aerr.Status())
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
	sendResponse(w, r, p)
}

func sendResponse(w http.ResponseWriter, r *http.Request, resp responder) {
	response := resp.ToJSON()

	enc := json.NewEncoder(w)
	if err := enc.Encode(&response); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func aclPermCheck(r *http.Request, org *organization.Organization, objACL *acl.ACL, perm string) util.Gerror {
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		return oerr
	}
	if f, ferr := objACL.CheckPerm(perm, opUser); ferr != nil {
		return ferr
	} else if !f {
		err := util.Errorf("You do not have permission to do that")
		err.SetStatus(http.StatusForbidden)
		return err
	}

	return nil
}

/*****************
 * Skeleton ACL handler functions

func _SKEL_ACLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "GET" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := _SKEL_.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	baseItemACLHandler(w, r, org, it)
}

func _SKEL_ACLPermHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)

	if r.Method != "PUT" {
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgName := vars["org"]
	org, orgerr := orgloader.Get(orgName)
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	it, iterr := _SKEL_.Get(org, vars["name"])
	if iterr != nil {
		jsonErrorReport(w, r, iterr.Error(), iterr.Status())
		return
	}
	perm := vars["perm"]

	baseACLPermHandler(w, r, org, it, perm)
}

******************/
