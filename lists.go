/* List handling stuff - a bit general, used by a few handlers */

/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
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
	"strings"
	"regexp"
)

func list_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")

	path_array := SplitPath(r.URL.Path)
	op := path_array[0]
	var list_data map[string]string
	switch op {
		case "nodes":
			list_data = node_handling(w, r)
		case "clients", "users":
			list_data = actor_handling(w, r, op)
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
	switch r.Method {
		case "GET":
			node_list := node.GetList()
			for _, k := range node_list {
				item_url := fmt.Sprintf("/nodes/%s", k)
				node_response[k] = util.CustomURL(item_url)
			}
		case "POST":
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
			chef_node.Save()
			node_response["uri"] = util.ObjURL(chef_node)
			w.WriteHeader(http.StatusCreated)
		default:
			JsonErrorReport(w, r, "Method not allowed for nodes", http.StatusMethodNotAllowed)
			return nil
	}
	return node_response
}

func actor_handling(w http.ResponseWriter, r *http.Request, op string) map[string]string {
	client_response := make(map[string]string)
	chef_type := strings.TrimSuffix(op, "s")
	switch r.Method {
		case "GET":
			client_list := actor.GetList()
			for _, k := range client_list {
				/* Make sure it's a client and not a user. */
				client_chk, _ := actor.Get(k)
				if client_chk.ChefType == chef_type {
					item_url := fmt.Sprintf("/%s/%s", op, k)
					client_response[k] = util.CustomURL(item_url)
				}
			}
		case "POST":
			client_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return nil
			}
			if name_val, ok := client_data["name"].(string); ok {
				client_chk, _ := actor.Get(name_val)
				if client_chk != nil {
					JsonErrorReport(w, r, "Client already exists", http.StatusConflict)
					return nil
				}
			} else {
					JsonErrorReport(w, r, "Field 'name' missing", http.StatusBadRequest)
					return nil
			}
			chef_client, err := actor.New(client_data["name"].(string), chef_type)
			if err != nil {
				var status int
				if m, _ := regexp.MatchString("Invalid client name.*", err.Error()); m {
					status = http.StatusBadRequest
				} else {
					status = http.StatusInternalServerError
				}
				JsonErrorReport(w, r, err.Error(), status)
				return nil
			}
			if admin_val, ok := client_data["admin"]; ok {
				switch admin_val := admin_val.(type){
					case bool:
						chef_client.Admin = admin_val
					default:
						JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
						return nil
				}
			} 

			if public_key, pkok := client_data["public_key"]; !pkok {
				if client_response["private_key"], err = chef_client.GenerateKeys(); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return nil
				}
			} else {
				switch public_key := public_key.(type) {
					case string:
						// TODO: validate public keys
						// when the time comes
						chef_client.PublicKey = public_key
					case nil:
						if client_response["private_key"], err = chef_client.GenerateKeys(); err != nil {
							JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
							return nil
						}
					default:
						JsonErrorReport(w, r, "Bad public key", http.StatusBadRequest)
						return nil
				}
			}
			/* If we make it here, we want the public key in the
			 * response. I think. */
			client_response["public_key"] = chef_client.PublicKey
			
			if validator_val, vok := client_data["validator"].(bool); vok && chef_type != "user"{
				chef_client.Validator = validator_val
			}
			chef_client.Save()
			client_response["uri"] = util.ObjURL(chef_client)
			w.WriteHeader(http.StatusCreated)
		default:
			JsonErrorReport(w, r, "Method not allowed for clients or users", http.StatusMethodNotAllowed)
			return nil
	}
	return client_response
}

func role_handling(w http.ResponseWriter, r *http.Request) map[string]string {
	role_response := make(map[string]string)
	switch r.Method {
		case "GET":
			role_list := role.GetList()
			for _, k := range role_list {
				item_url := fmt.Sprintf("/roles/%s", k)
				role_response[k] = util.CustomURL(item_url)
			}
		case "POST":
			role_data, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			}
			chef_role, err := role.Get(role_data["name"].(string))
			if chef_role != nil {
				httperr := fmt.Errorf("Role %s already exists.", role_data["name"].(string))
				JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
				return nil
			}
			chef_role, err = role.NewFromJson(role_data)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return nil
			}
			chef_role.Save()
			role_response["uri"] = util.ObjURL(chef_role)
			w.WriteHeader(http.StatusCreated)
		default:
			JsonErrorReport(w, r, "Method not allowed for roles", http.StatusMethodNotAllowed)
			return nil
	}
	return role_response
}
