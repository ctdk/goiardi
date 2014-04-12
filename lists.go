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
	"fmt"
	"net/http"
	"encoding/json"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/user"
)

func list_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")

	path_array := SplitPath(r.URL.Path)
	op := path_array[0]
	var list_data map[string]string
	switch op {
		case "nodes":
			list_data = node_handling(w, r)
		case "clients":
			list_data = client_handling(w, r)
		case "users":
			list_data = user_handling(w, r)
		case "roles":
			list_data = role_handling(w, r)
		default:
			list_data = make(map[string]string)
			list_data["huh"] = "not valid"
	}
	if list_data != nil {
		enc := json.NewEncoder(w)
		if err := enc.Encode(&list_data); err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
	}
}

func node_handling(w http.ResponseWriter, r *http.Request) map[string]string {
	/* We're dealing with nodes, then. */
	node_response := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}
	switch r.Method {
		case "GET":
			if opUser.IsValidator() {
				JsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return nil
			}
			node_list := node.GetList()
			for _, k := range node_list {
				item_url := fmt.Sprintf("/nodes/%s", k)
				node_response[k] = util.CustomURL(item_url)
			}
		case "POST":
			if opUser.IsValidator() {
				JsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return nil
			}
			node_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return nil
			}
			node_name, sterr := util.ValidateAsString(node_data["name"])
			if sterr != nil {
				JsonErrorReport(w, r, sterr.Error(), http.StatusBadRequest)
				return nil
			}
			chef_node, _ := node.Get(node_name)
			if chef_node != nil {
				httperr := fmt.Errorf("Node already exists")
				JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
				return nil
			}
			var nerr util.Gerror
			chef_node, nerr = node.NewFromJson(node_data)
			if nerr != nil {
				JsonErrorReport(w, r, nerr.Error(), nerr.Status())
				return nil
			}
			err := chef_node.Save()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return nil
			}
			node_response["uri"] = util.ObjURL(chef_node)
			w.WriteHeader(http.StatusCreated)
		default:
			JsonErrorReport(w, r, "Method not allowed for nodes", http.StatusMethodNotAllowed)
			return nil
	}
	return node_response
}

func client_handling(w http.ResponseWriter, r *http.Request) map[string]string {
	client_response := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}

	switch r.Method {
		case "GET":
			client_list := client.GetList()
			for _, k := range client_list {
				/* Make sure it's a client and not a user. */
				item_url := fmt.Sprintf("/clients/%s", k)
				client_response[k] = util.CustomURL(item_url)
			}
		case "POST":
			client_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return nil
			}
			if averr := util.CheckAdminPlusValidator(client_data); averr != nil {
				JsonErrorReport(w, r, averr.Error(), averr.Status())
				return nil
			}
			if !opUser.IsAdmin() && !opUser.IsValidator() {
				JsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return nil
			} else if !opUser.IsAdmin() && opUser.IsValidator() {
				if aerr := opUser.CheckPermEdit(client_data, "admin"); aerr != nil {
					JsonErrorReport(w, r, aerr.Error(), aerr.Status())
					return nil
				}
				if verr := opUser.CheckPermEdit(client_data, "validator"); verr != nil {
					JsonErrorReport(w, r, verr.Error(), verr.Status())
					return nil
				}

			}
			client_name, sterr := util.ValidateAsString(client_data["name"])
			if sterr != nil || client_name == "" {
				err := fmt.Errorf("Field 'name' missing")
				JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return nil
			}

			chef_client, err := client.NewFromJson(client_data)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), err.Status())
				return nil
			}

			if public_key, pkok := client_data["public_key"]; !pkok {
				var perr error
				if client_response["private_key"], perr = chef_client.GenerateKeys(); perr != nil {
					JsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
					return nil
				}
			} else {
				switch public_key := public_key.(type) {
					case string:
						if pkok, pkerr := client.ValidatePublicKey(public_key); !pkok {
							JsonErrorReport(w, r, pkerr.Error(), pkerr.Status())
							return nil
						}
						chef_client.SetPublicKey(public_key)
					case nil:
			
						var perr error
						if client_response["private_key"], perr = chef_client.GenerateKeys(); perr != nil {
							JsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
							return nil
						}
					default:
						JsonErrorReport(w, r, "Bad public key", http.StatusBadRequest)
						return nil
				}
			}
			/* If we make it here, we want the public key in the
			 * response. I think. */
			client_response["public_key"] = chef_client.PublicKey()
			
			chef_client.Save()
			client_response["uri"] = util.ObjURL(chef_client)
			w.WriteHeader(http.StatusCreated)
		default:
			JsonErrorReport(w, r, "Method not allowed for clients or users", http.StatusMethodNotAllowed)
			return nil
	}
	return client_response
}

// user handling
func user_handling(w http.ResponseWriter, r *http.Request) map[string]string {
	user_response := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}

	switch r.Method {
		case "GET":
			user_list := user.GetList()
			for _, k := range user_list {
				/* Make sure it's a client and not a user. */
				item_url := fmt.Sprintf("/users/%s", k)
				user_response[k] = util.CustomURL(item_url)
			}
		case "POST":
			user_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return nil
			}
			if averr := util.CheckAdminPlusValidator(user_data); averr != nil {
				JsonErrorReport(w, r, averr.Error(), averr.Status())
				return nil
			}
			if !opUser.IsAdmin() && !opUser.IsValidator() {
				JsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return nil
			} else if !opUser.IsAdmin() && opUser.IsValidator() {
				if aerr := opUser.CheckPermEdit(user_data, "admin"); aerr != nil {
					JsonErrorReport(w, r, aerr.Error(), aerr.Status())
					return nil
				}
				if verr := opUser.CheckPermEdit(user_data, "validator"); verr != nil {
					JsonErrorReport(w, r, verr.Error(), verr.Status())
					return nil
				}

			}
			user_name, sterr := util.ValidateAsString(user_data["name"])
			if sterr != nil || user_name == "" {
				err := fmt.Errorf("Field 'name' missing")
				JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return nil
			}

			chef_user, err := user.NewFromJson(user_data)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), err.Status())
				return nil
			}

			if public_key, pkok := user_data["public_key"]; !pkok {
				var perr error
				if user_response["private_key"], perr = chef_user.GenerateKeys(); perr != nil {
					JsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
					return nil
				}
			} else {
				switch public_key := public_key.(type) {
					case string:
						if pkok, pkerr := user.ValidatePublicKey(public_key); !pkok {
							JsonErrorReport(w, r, pkerr.Error(), pkerr.Status())
							return nil
						}
						chef_user.SetPublicKey(public_key)
					case nil:
			
						var perr error
						if user_response["private_key"], perr = chef_user.GenerateKeys(); perr != nil {
							JsonErrorReport(w, r, perr.Error(), http.StatusInternalServerError)
							return nil
						}
					default:
						JsonErrorReport(w, r, "Bad public key", http.StatusBadRequest)
						return nil
				}
			}
			/* If we make it here, we want the public key in the
			 * response. I think. */
			user_response["public_key"] = chef_user.PublicKey()
			
			chef_user.Save()
			user_response["uri"] = util.ObjURL(chef_user)
			w.WriteHeader(http.StatusCreated)
		default:
			JsonErrorReport(w, r, "Method not allowed for clients or users", http.StatusMethodNotAllowed)
			return nil
	}
	return user_response
}

func role_handling(w http.ResponseWriter, r *http.Request) map[string]string {
	role_response := make(map[string]string)
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return nil
	}
	switch r.Method {
		case "GET":
			if opUser.IsValidator() {
				JsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return nil
			}
			role_list := role.GetList()
			for _, k := range role_list {
				item_url := fmt.Sprintf("/roles/%s", k)
				role_response[k] = util.CustomURL(item_url)
			}
		case "POST":
			if !opUser.IsAdmin() {
				JsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
				return nil
			}
			role_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return nil
			}
			if _, ok := role_data["name"].(string); !ok {
				JsonErrorReport(w, r, "Role name missing", http.StatusBadRequest)
				return nil
			}
			chef_role, _ := role.Get(role_data["name"].(string))
			if chef_role != nil {
				httperr := fmt.Errorf("Role already exists")
				JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
				return nil
			}
			var nerr util.Gerror
			chef_role, nerr = role.NewFromJson(role_data)
			if nerr != nil {
				JsonErrorReport(w, r, nerr.Error(), nerr.Status())
				return nil
			}
			err := chef_role.Save()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return nil
			}
			role_response["uri"] = util.ObjURL(chef_role)
			w.WriteHeader(http.StatusCreated)
		default:
			JsonErrorReport(w, r, "Method not allowed for roles", http.StatusMethodNotAllowed)
			return nil
	}
	return role_response
}
