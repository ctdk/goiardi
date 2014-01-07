/* Environment functions */

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
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"fmt"
	"encoding/json"
	"strings"
)

func environment_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	accErr := CheckAccept(w, r, "application/json")
	if accErr != nil {
		JsonErrorReport(w, r, accErr.Error(), http.StatusNotAcceptable)
		return
	}
	path_array := SplitPath(r.URL.Path)
	env_response := make(map[string]interface{})
	// num_results := r.FormValue("num_versions")
	var num_results string
	r.ParseForm()
	if nrs, found := r.Form["num_versions"]; found {
		if len(nrs) < 0 {
			JsonErrorReport(w, r, "invalid num_versions", http.StatusBadRequest)
			return
		}
		num_results = nrs[0]
		err := util.ValidateNumVersions(num_results)
		if err != nil {
			JsonErrorReport(w, r, "You have requested an invalid number of versions (x >= 0 || 'all')", err.Status())
			return
		}
	}

	path_array_len := len(path_array)

	if path_array_len == 1 {
		switch r.Method {
			case "GET":
				env_list := environment.GetList()
				for _, env := range env_list {
					item_url := fmt.Sprintf("/environments/%s", env)
					env_response[env] = util.CustomURL(item_url)
				}
			case "POST":
				env_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
					return
				}
				if _, ok := env_data["name"].(string); !ok || env_data["name"].(string) == "" {
					JsonErrorReport(w, r, "Environment name missing", http.StatusBadRequest)
					return
				}
				chef_env, _ := environment.Get(env_data["name"].(string))
				if chef_env != nil {
					httperr := fmt.Errorf("Environment already exists")
					JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
					return
				}
				var eerr util.Gerror
				chef_env, eerr = environment.NewFromJson(env_data)
				if eerr != nil {
					JsonErrorReport(w, r, eerr.Error(), eerr.Status())
					return
				}
				if err := chef_env.Save(); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
				env_response["uri"] = util.ObjURL(chef_env)
				w.WriteHeader(http.StatusCreated)
			default:
				JsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
				return
		}
	} else if path_array_len == 2 {
		/* All of the 2 element operations return the environment
		 * object, so we do the json encoding in this block and return 
		 * out. */
		env_name := path_array[1]
		env, err := environment.Get(env_name)
		del_env := false /* Set this to delete the environment after
				  * sending the json. */
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		switch r.Method {
			case "GET", "DELETE":
				/* We don't actually have to do much here. */
				if r.Method == "DELETE" {
					if env_name == "_default" {
						JsonErrorReport(w, r, "The '_default' environment cannot be modified.", http.StatusMethodNotAllowed)
						return	
					} else {
						del_env = true
					}
				}
			case "PUT":
				env_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
					return
				}
				if env_data == nil {
					JsonErrorReport(w, r, "No environment data in body at all!", http.StatusBadRequest)
					return
				}
				if _, ok := env_data["name"]; !ok {
					//env_data["name"] = env_name
					JsonErrorReport(w, r, "Environment name missing", http.StatusBadRequest)
					return
				}
				json_name, sterr := util.ValidateAsString(env_data["name"])
				if sterr != nil {
					JsonErrorReport(w, r, sterr.Error(), sterr.Status())
					return
				} else if json_name == "" {
					JsonErrorReport(w, r, "Environment name missing", http.StatusBadRequest)
					return
				}
				if env_name != env_data["name"].(string) {
					env, err = environment.Get(env_data["name"].(string))
					if err == nil {
						JsonErrorReport(w, r, "Environment already exists", http.StatusConflict)
						return
					} else {
						var eerr util.Gerror
						env, eerr = environment.NewFromJson(env_data)
						if eerr != nil {
							JsonErrorReport(w, r, eerr.Error(), eerr.Status())
							return
						}
						w.WriteHeader(http.StatusCreated)
						oldenv, olderr := environment.Get(env_name)
						if olderr == nil {
							oldenv.Delete()
						}
					}
				} else {
					if json_name == "" {
						env_data["name"] = env_name
					}
					if err := env.UpdateFromJson(env_data); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
						return
					}
				}
				if err := env.Save(); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
			default:
				JsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
				return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(&env); err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if del_env {
			err = env.Delete()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		return
	} else if path_array_len == 3 {
		env_name := path_array[1]
		op := path_array[2]

		if op == "cookbook_versions" && r.Method != "POST" || op != "cookbook_versions" && r.Method != "GET" {
			JsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}

		env, err := environment.Get(env_name)
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		switch op {
			case "cookbook_versions":
				/* Chef Server API docs aren't even remotely
				 * right here. What it actually wants is the
				 * usual hash of info for the latest or
				 * constrained version. Weird. */
				cb_ver, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					errmsg := jerr.Error()
					if !strings.Contains(errmsg, "Field") {
						errmsg = "invalid JSON"
					} else {
						errmsg = jerr.Error()
					}
					JsonErrorReport(w, r, errmsg, http.StatusBadRequest)
					return
				}

				if _, ok := cb_ver["run_list"]; !ok {
					JsonErrorReport(w, r, "POSTed JSON badly formed.", http.StatusMethodNotAllowed)
					return
				}
				deps, err := cookbook.DependsCookbooks(cb_ver["run_list"].([]string), env.CookbookVersions)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusPreconditionFailed)
					return
				}
				/* Need our own encoding here too. */
				enc := json.NewEncoder(w)
				if err := enc.Encode(&deps); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				}
				return
			case "cookbooks":
				env_response = env.AllCookbookHash(num_results)
			case "nodes":
				node_list := node.GetList()
				for _, n := range node_list {
					chef_node, _ := node.Get(n)
					if chef_node == nil {
						continue
					}
					if chef_node.ChefEnvironment == env_name {
						env_response[chef_node.Name] = util.ObjURL(chef_node) 
					}
				}
			case "recipes":
				env_recipes := env.RecipeList()
				/* And... we have to do our own json response
				 * here. Hmph. */
				/* TODO: make the JSON encoding stuff its own
				 * function. Dunno why I never thought of that
				 * before now for this. */
				enc := json.NewEncoder(w)
				if err := enc.Encode(&env_recipes); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				}
				return
			default:
				JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return

		}
	} else if path_array_len == 4 {
		env_name := path_array[1]
		/* op is either "cookbooks" or "roles", and op_name is the name
		 * of the object op refers to. */
		op := path_array[2]
		op_name := path_array[3]

		if r.Method != "GET" {
			JsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		/* Redirect op=roles to /roles/NAME/environments/NAME. The API
		 * docs recommend but do not require using that URL, so in the
		 * interest of simplicity we will just redirect to it. */
		if op == "roles" {
			redir_url := fmt.Sprintf("/roles/%s/environments/%s", op_name, env_name)
			http.Redirect(w, r, redir_url, http.StatusMovedPermanently)
			return
		} else if op == "cookbooks" {
			env, err := environment.Get(env_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			cb, err := cookbook.Get(op_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			/* Here and, I think, here only, if num_versions isn't
			 * set it's supposed to return ALL matching versions.
			 * API docs are wrong here. */
			if num_results == "" {
				num_results = "all"
			}
			env_response[op_name] = cb.ConstrainedInfoHash(num_results, env.CookbookVersions[op_name])
		} else {
			/* Not an op we know. */
			JsonErrorReport(w, r, "Bad request - too many elements in path", http.StatusBadRequest)
			return
		}
	} else {
		/* Bad number of path elements. */
		JsonErrorReport(w, r, "Bad request - too many elements in path", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&env_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
