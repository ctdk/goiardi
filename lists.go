/* List handling stuff - a bit general, used by a few handlers */

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
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	pathArray := splitPath(r.URL.Path)
	op := pathArray[2]
	var listData map[string]string
	switch op {
	case "nodes":
		listData = nodeHandling(org, w, r)
	case "clients":
		listData = clientHandling(org, w, r)
	case "roles":
		listData = roleHandling(org, w, r)
	default:
		listData = make(map[string]string)
		listData["huh"] = "not valid"
	}
	if listData != nil {
		enc := json.NewEncoder(w)
		if err := enc.Encode(&listData); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
	}
}

func nodeHandling(org *organization.Organization, w http.ResponseWriter, r *http.Request) map[string]string {
	/* We're dealing with nodes, then. */
	nodeResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}
	switch r.Method {
	case "GET":
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		nodeList := node.GetList(org)
		for _, k := range nodeList {
			itemURL := util.JoinStr("/organizations/", org.Name, "/nodes/", k)
			nodeResponse[k] = util.CustomURL(itemURL)
		}
	case "POST":
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		nodeData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return nil
		}
		nodeName, sterr := util.ValidateAsString(nodeData["name"])
		if sterr != nil {
			jsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
			return nil
		}
		chefNode, _ := node.Get(org, nodeName)
		if chefNode != nil {
			httperr := fmt.Errorf("Node already exists")
			jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
			return nil
		}
		var nerr util.Gerror
		chefNode, nerr = node.NewFromJSON(org, nodeData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return nil
		}
		err := chefNode.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return nil
		}
		err = chefNode.UpdateStatus("new")
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return nil
		}
		if lerr := loginfo.LogEvent(org, opUser, chefNode, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return nil
		}
		nodeResponse["uri"] = util.ObjURL(chefNode)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for nodes", http.StatusMethodNotAllowed)
		return nil
	}
	return nodeResponse
}

func clientHandling(org *organization.Organization, w http.ResponseWriter, r *http.Request) map[string]string {
	clientResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}

	switch r.Method {
	case "GET":
		clientList := client.GetList(org)
		for _, k := range clientList {
			/* Make sure it's a client and not a user. */
			itemURL := util.JoinStr("/organizations/", org.Name, "/clients/", k)
			clientResponse[k] = util.CustomURL(itemURL)
		}
	case "POST":
		clientData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return nil
		}
		if averr := util.CheckAdminPlusValidator(clientData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return nil
		}
		/*
		if !opUser.IsAdmin() && !opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		} else if !opUser.IsAdmin() && opUser.IsValidator() {
			if aerr := opUser.CheckPermEdit(clientData, "admin"); aerr != nil {
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return nil
			}
			if verr := opUser.CheckPermEdit(clientData, "validator"); verr != nil {
				jsonErrorReport(w, r, verr.Error(), verr.Status())
				return nil
			}

		}
		*/
		clientACL, err := acl.Get(org, "container", "client")
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return nil
		} 
		if f, err := clientACL.CheckPerm("create", opUser); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return nil
		} else if !f {
			jsonErrorReport(w, r, "You are not allowed to perform that action", http.StatusForbidden)
			return nil
		}
		clientName, sterr := util.ValidateAsString(clientData["name"])
		if sterr != nil || clientName == "" {
			err := fmt.Errorf("Field 'name' missing")
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return nil
		}

		chefClient, err := client.NewFromJSON(org, clientData)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return nil
		}

		if publicKey, pkok := clientData["public_key"]; !pkok {
			var perr error
			if clientResponse["private_key"], perr = chefClient.GenerateKeys(); perr != nil {
				jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
				return nil
			}
		} else {
			switch publicKey := publicKey.(type) {
			case string:
				if pkok, pkerr := client.ValidatePublicKey(publicKey); !pkok {
					jsonErrorReport(w, r, pkerr.Error(), pkerr.Status())
					return nil
				}
				chefClient.SetPublicKey(publicKey)
			case nil:

				var perr error
				if clientResponse["private_key"], perr = chefClient.GenerateKeys(); perr != nil {
					jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
					return nil
				}
			default:
				jsonErrorReport(w, r, "Bad public key", http.StatusBadRequest)
				return nil
			}
		}
		/* If we make it here, we want the public key in the
		 * response. I think. */
		clientResponse["public_key"] = chefClient.PublicKey()

		chefClient.Save()
		if lerr := loginfo.LogEvent(org, opUser, chefClient, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return nil
		}
		clientResponse["uri"] = util.ObjURL(chefClient)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for clients or users", http.StatusMethodNotAllowed)
		return nil
	}
	return clientResponse
}

func roleHandling(org *organization.Organization, w http.ResponseWriter, r *http.Request) map[string]string {
	roleResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}
	switch r.Method {
	case "GET":
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		roleList := role.GetList(org)
		for _, k := range roleList {
			itemURL := util.JoinStr("/organizations/", org.Name, "/roles/", k)
			roleResponse[k] = util.CustomURL(itemURL)
		}
	case "POST":
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		roleData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return nil
		}
		if _, ok := roleData["name"].(string); !ok {
			jsonErrorReport(w, r, "Role name missing", http.StatusBadRequest)
			return nil
		}
		chefRole, _ := role.Get(org, roleData["name"].(string))
		if chefRole != nil {
			httperr := fmt.Errorf("Role already exists")
			jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
			return nil
		}
		var nerr util.Gerror
		chefRole, nerr = role.NewFromJSON(org, roleData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return nil
		}
		err := chefRole.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return nil
		}
		if lerr := loginfo.LogEvent(org, opUser, chefRole, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return nil
		}
		roleResponse["uri"] = util.ObjURL(chefRole)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for roles", http.StatusMethodNotAllowed)
		return nil
	}
	return roleResponse
}
