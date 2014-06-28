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
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/log_info"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"strings"
)

func environment_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accErr := CheckAccept(w, r, "application/json")
	if accErr != nil {
		JsonErrorReport(w, r, accErr.Error(), http.StatusNotAcceptable)
		return
	}

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
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
			if opUser.IsValidator() {
				JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			env_list := environment.GetList()
			for _, env := range env_list {
				item_url := fmt.Sprintf("/environments/%s", env)
				env_response[env] = util.CustomURL(item_url)
			}
		case "POST":
			if !opUser.IsAdmin() {
				JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
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
			if lerr := log_info.LogEvent(opUser, chef_env, "create"); lerr != nil {
				JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
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
				if !opUser.IsAdmin() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				if env_name == "_default" {
					JsonErrorReport(w, r, "The '_default' environment cannot be modified.", http.StatusMethodNotAllowed)
					return
				} else {
					del_env = true
				}
			} else {
				if opUser.IsValidator() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
			}
		case "PUT":
			if !opUser.IsAdmin() {
				JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
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
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			}
			if err := env.Save(); err != nil {
				JsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			if lerr := log_info.LogEvent(opUser, env, "modify"); lerr != nil {
				JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
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
			err := env.Delete()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := log_info.LogEvent(opUser, env, "delete"); lerr != nil {
				JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
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

		if opUser.IsValidator() {
			JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
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
			node_list, err := node.GetFromEnv(env_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, chef_node := range node_list {
				env_response[chef_node.Name] = util.ObjURL(chef_node)
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
		if opUser.IsValidator() {
			JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		env, err := environment.Get(env_name)
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		/* Biting the bullet and not redirecting this to
		 * /roles/NAME/environments/NAME. The behavior is exactly the
		 * same, but it makes clients and chef-pedant somewhat unhappy
		 * to not have this way available. */
		if op == "roles" {
			role, err := role.Get(op_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			var run_list []string
			if env_name == "_default" {
				run_list = role.RunList
			} else {
				run_list = role.EnvRunLists[env_name]
			}
			env_response["run_list"] = run_list
		} else if op == "cookbooks" {
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
