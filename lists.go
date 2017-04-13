/* List handling stuff - a bit general, used by a few handlers */

/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pathArray := splitPath(r.URL.Path)
	op := pathArray[0]

	// if we somehow get a HEAD req to one of these, where they aren't
	// all that meaningful, just return StatusOK until we get word that we
	// ought to do something else
	//
	// Mercifully this weird set of functions is out in 1.0.0-dev

	if r.Method == http.MethodHead {
		switch op {
		case "nodes", "clients", "users", "roles":
			headDefaultResponse(w, r)
		default:
			headResponse(w, r, http.StatusInternalServerError)
		}
		return
	}
	
	var listData map[string]string
	switch op {
	case "nodes":
		listData = nodeHandling(w, r)
	case "clients":
		listData = clientHandling(w, r)
	case "users":
		listData = userHandling(w, r)
	case "roles":
		listData = roleHandling(w, r)
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

func nodeHandling(w http.ResponseWriter, r *http.Request) map[string]string {
	/* We're dealing with nodes, then. */
	nodeResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}
	switch r.Method {
	case http.MethodGet:
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		nodeList := node.GetList()
		for _, k := range nodeList {
			itemURL := fmt.Sprintf("/nodes/%s", k)
			nodeResponse[k] = util.CustomURL(itemURL)
		}
	case http.MethodPost:
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
		chefNode, _ := node.Get(nodeName)
		if chefNode != nil {
			httperr := fmt.Errorf("Node already exists")
			jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
			return nil
		}
		var nerr util.Gerror
		chefNode, nerr = node.NewFromJSON(nodeData)
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
		if lerr := loginfo.LogEvent(opUser, chefNode, "create"); lerr != nil {
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

func clientHandling(w http.ResponseWriter, r *http.Request) map[string]string {
	clientResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}

	switch r.Method {
	case http.MethodGet:
		clientList := client.GetList()
		for _, k := range clientList {
			/* Make sure it's a client and not a user. */
			itemURL := fmt.Sprintf("/clients/%s", k)
			clientResponse[k] = util.CustomURL(itemURL)
		}
	case http.MethodPost:
		clientData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return nil
		}
		if averr := util.CheckAdminPlusValidator(clientData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return nil
		}
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
		clientName, sterr := util.ValidateAsString(clientData["name"])
		if sterr != nil || clientName == "" {
			err := fmt.Errorf("Field 'name' missing")
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return nil
		}

		chefClient, err := client.NewFromJSON(clientData)
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

		err = chefClient.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return nil
		}

		if lerr := loginfo.LogEvent(opUser, chefClient, "create"); lerr != nil {
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

// user handling
func userHandling(w http.ResponseWriter, r *http.Request) map[string]string {
	userResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}

	switch r.Method {
	case http.MethodGet:
		userList := user.GetList()
		for _, k := range userList {
			/* Make sure it's a client and not a user. */
			itemURL := fmt.Sprintf("/users/%s", k)
			userResponse[k] = util.CustomURL(itemURL)
		}
	case http.MethodPost:
		userData, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return nil
		}
		if averr := util.CheckAdminPlusValidator(userData); averr != nil {
			jsonErrorReport(w, r, averr.Error(), averr.Status())
			return nil
		}
		if !opUser.IsAdmin() && !opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		} else if !opUser.IsAdmin() && opUser.IsValidator() {
			if aerr := opUser.CheckPermEdit(userData, "admin"); aerr != nil {
				jsonErrorReport(w, r, aerr.Error(), aerr.Status())
				return nil
			}
			if verr := opUser.CheckPermEdit(userData, "validator"); verr != nil {
				jsonErrorReport(w, r, verr.Error(), verr.Status())
				return nil
			}

		}
		userName, sterr := util.ValidateAsString(userData["name"])
		if sterr != nil || userName == "" {
			err := fmt.Errorf("Field 'name' missing")
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return nil
		}

		chefUser, err := user.NewFromJSON(userData)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return nil
		}

		if publicKey, pkok := userData["public_key"]; !pkok {
			var perr error
			if userResponse["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
				jsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
				return nil
			}
		} else {
			switch publicKey := publicKey.(type) {
			case string:
				if pkok, pkerr := user.ValidatePublicKey(publicKey); !pkok {
					jsonErrorReport(w, r, pkerr.Error(), pkerr.Status())
					return nil
				}
				chefUser.SetPublicKey(publicKey)
			case nil:

				var perr error
				if userResponse["private_key"], perr = chefUser.GenerateKeys(); perr != nil {
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
		userResponse["public_key"] = chefUser.PublicKey()

		chefUser.Save()
		if lerr := loginfo.LogEvent(opUser, chefUser, "create"); lerr != nil {
			jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
			return nil
		}
		userResponse["uri"] = util.ObjURL(chefUser)
		w.WriteHeader(http.StatusCreated)
	default:
		jsonErrorReport(w, r, "Method not allowed for clients or users", http.StatusMethodNotAllowed)
		return nil
	}
	return userResponse
}

func roleHandling(w http.ResponseWriter, r *http.Request) map[string]string {
	roleResponse := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}
	switch r.Method {
	case http.MethodGet:
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return nil
		}
		roleList := role.GetList()
		for _, k := range roleList {
			itemURL := fmt.Sprintf("/roles/%s", k)
			roleResponse[k] = util.CustomURL(itemURL)
		}
	case http.MethodPost:
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
		chefRole, _ := role.Get(roleData["name"].(string))
		if chefRole != nil {
			httperr := fmt.Errorf("Role already exists")
			jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
			return nil
		}
		var nerr util.Gerror
		chefRole, nerr = role.NewFromJSON(roleData)
		if nerr != nil {
			jsonErrorReport(w, r, nerr.Error(), nerr.Status())
			return nil
		}
		err := chefRole.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return nil
		}
		if lerr := loginfo.LogEvent(opUser, chefRole, "create"); lerr != nil {
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
