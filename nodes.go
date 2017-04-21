/* Node functions */

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
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func nodeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	nodeName := vars["name"]

	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if opUser.IsValidator() {
		jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
		return
	}

	// leave the HEAD check out of the switch below, so we can avoid getting
	// the entire node when we don't need it.
	if r.Method == http.MethodHead {
		permCheck := func(r *http.Request, nodeName string, opUser actor.Actor) util.Gerror {
			if opUser.IsValidator() {
				return headForbidden()
			}
			return nil
		}
		headChecking(w, r, opUser, org, nodeName, node.DoesExist, permCheck)
		return
	}

	chefNode, nerr := node.Get(org, nodeName)
	if nerr != nil {
		jsonErrorReport(w, r, nerr.Error(), http.StatusNotFound)
		return
	}

	containerACL, err := acl.GetItemACL(org, chefNode)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}

	/* So, what are we doing? Depends on the HTTP method, of course */
	switch r.Method {
	case http.MethodGet, http.MethodDelete:
		if r.Method == "DELETE" {
			delchk, ferr := containerACL.CheckPerm("delete", opUser)
			if ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			}
			if !delchk && !(opUser.IsClient() && opUser.(*client.Client).NodeName == nodeName) {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
		} else {
			if f, ferr := containerACL.CheckPerm("read", opUser); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
				return
			}
		}

		enc := json.NewEncoder(w)
		if err := enc.Encode(&chefNode); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodDelete {
			err := chefNode.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if aerr := containerACL.Delete(); aerr != nil {
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return
			}
			if lerr := loginfo.LogEvent(org, opUser, chefNode, "delete"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
		}
	case http.MethodPut:
		updatechk, ferr := containerACL.CheckPerm("update", opUser)
		if ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		}
		if !updatechk && !(opUser.IsClient() && opUser.(*client.Client).NodeName == nodeName) {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		nodeData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		/* If nodeName and nodeData["name"] don't match, we
		 * need to make a new node. Make sure that node doesn't
		 * exist. */
		if _, found := nodeData["name"]; !found {
			nodeData["name"] = nodeName
		}
		jsonName, sterr := util.ValidateAsString(nodeData["name"])
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}
		if nodeName != jsonName && jsonName != "" {
			jsonErrorReport(w, r, "Node name mismatch.", http.StatusBadRequest)
			return
		}
		if jsonName == "" {
			nodeData["name"] = nodeName
		}
		nerr := chefNode.UpdateFromJSON(nodeData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return
		}
		err := chefNode.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if lerr := loginfo.LogEvent(org, opUser, chefNode, "modify"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(&chefNode); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
	}
}

func nodeListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	nodeResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	containerACL, err := acl.GetContainerACL(org, "nodes")
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
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
		nodeList := node.GetList(org)
		for _, k := range nodeList {
			itemURL := util.JoinStr("/organizations/", org.Name, "/nodes/", k)
			nodeResponse[k] = util.CustomURL(itemURL)
		}
	case "POST":
		if f, ferr := containerACL.CheckPerm("create", opUser); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		nodeData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		nodeName, sterr := util.ValidateAsString(nodeData["name"])
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return
		}
		chefNode, _ := node.Get(org, nodeName)
		if chefNode != nil {
			httperr := fmt.Errorf("Node already exists")
			jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
			return
		}
		var nerr util.Gerror
		chefNode, nerr = node.NewFromJSON(org, nodeData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return
		}
		err := chefNode.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		err = chefNode.UpdateStatus("new")
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if lerr := loginfo.LogEvent(org, opUser, chefNode, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return
		}
		nodeResponse["uri"] = util.ObjURL(chefNode)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for nodes", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&nodeResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
	return
}
