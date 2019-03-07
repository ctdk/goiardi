/* Environment functions */

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
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"github.com/tideland/golib/logger"
	"net/http"
	"strings"
)

func environmentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	accErr := checkAccept(w, r, "application/json")
	if accErr != nil {
		jsonErrorReport(w, r, accErr.Error(), http.StatusNotAcceptable)
		return
	}

	vars := mux.Vars(r)
	org, orgerr := reqctx.CtxOrg(r.Context())
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	pathArray := splitPath(r.URL.Path)[2:]
	envResponse := make(map[string]interface{})
	var numResults string
	r.ParseForm()
	if nrs, found := r.Form["num_versions"]; found {
		if len(nrs) < 0 {
			jsonErrorReport(w, r, "invalid num_versions", http.StatusBadRequest)
			return
		}
		numResults = nrs[0]
		err := util.ValidateNumVersions(numResults)
		if err != nil {
			jsonErrorReport(w, r, "You have requested an invalid number of versions (x >= 0 || 'all')", err.Status())
			return
		}
	}

	pathArrayLen := len(pathArray)

	if pathArrayLen == 1 {
		// Somewhat weirdly, users can create environments even if they
		// can't read them.
		if r.Method != "POST" {
			if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "environments", "read"); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
				return
			}
		}
		switch r.Method {
		case http.MethodHead:
			// not real meaningful...
			if opUser.IsValidator() {
				headResponse(w, r, http.StatusForbidden)
				return
			}
			headDefaultResponse(w, r)
			return
		case http.MethodGet:
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			envList := environment.GetList(org)
			for _, env := range envList {
				envResponse[env] = util.CustomURL(util.JoinStr("/organizations/", org.Name, "/environments/", env))
			}
		case http.MethodPost:
			if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "environments", "create"); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You are not allowed to perform this action.", http.StatusForbidden)
				return
			}
			envData, jerr := parseObjJSON(r.Body)
			if jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			if _, ok := envData["name"].(string); !ok || envData["name"].(string) == "" {
				jsonErrorReport(w, r, "Environment name missing", http.StatusBadRequest)
				return
			}
			chefEnv, _ := environment.Get(org, envData["name"].(string))
			if chefEnv != nil {
				httperr := fmt.Errorf("Environment already exists")
				jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
				return
			}
			var eerr util.Gerror
			chefEnv, eerr = environment.NewFromJSON(org, envData)
			if eerr != nil {
				jsonErrorReport(w, r, eerr.Error(), eerr.Status())
				return
			}
			if err := chefEnv.Save(); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return
			}

			// creator perms
			if err := org.PermCheck.EditItemPerm(chefEnv, opUser, aclhelper.DefaultACLs[:], "add"); err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			if lerr := loginfo.LogEvent(org, opUser, chefEnv, "create"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
			envResponse["uri"] = util.ObjURL(chefEnv)
			w.WriteHeader(http.StatusCreated)
		default:
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
	} else if pathArrayLen == 2 {
		/* All of the 2 element operations return the environment
		 * object, so we do the json encoding in this block and return
		 * out. */

		envName := vars["name"]

		// get the HEAD out of the way before doing heavy lifting with
		// environments
		if r.Method == http.MethodHead {
			permCheck := func(r *http.Request, envName string, opUser actor.Actor) util.Gerror {
				if opUser.IsValidator() {
					return headForbidden()
				}
				return nil
			}
			headChecking(w, r, opUser, org, envName, environment.DoesExist, permCheck)
			return
		}

		env, err := environment.Get(org, envName)
		delEnv := false /* Set this to delete the environment after
		 * sending the json. */
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet, http.MethodDelete:
			/* We don't actually have to do much here. */
			if r.Method == http.MethodDelete {
				if f, ferr := org.PermCheck.CheckItemPerm(env, opUser, "delete"); ferr != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else if !f {
					jsonErrorReport(w, r, "You are not allowed to perform this action.", http.StatusForbidden)
					return
				}
				if envName == "_default" {
					jsonErrorReport(w, r, "The '_default' environment cannot be modified.", http.StatusMethodNotAllowed)
					return
				}
				delEnv = true
			} else {
				if f, ferr := org.PermCheck.CheckItemPerm(env, opUser, "read"); ferr != nil {
					jsonErrorReport(w, r, ferr.Error(), ferr.Status())
					return
				} else if !f {
					jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
					return
				}
				if opUser.IsValidator() {
					jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
			}
		case http.MethodPut:
			if f, ferr := org.PermCheck.CheckItemPerm(env, opUser, "update"); ferr != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You are not allowed to perform this action.", http.StatusForbidden)
				return
			}
			envData, jerr := parseObjJSON(r.Body)
			if jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			if envData == nil {
				jsonErrorReport(w, r, "No environment data in body at all!", http.StatusBadRequest)
				return
			}
			if _, ok := envData["name"]; !ok {
				//envData["name"] = envName
				jsonErrorReport(w, r, "Environment name missing", http.StatusBadRequest)
				return
			}
			jsonName, sterr := util.ValidateAsString(envData["name"])
			if sterr != nil {
				jsonErrorReport(w, r, sterr.Error(), sterr.Status())
				return
			} else if jsonName == "" {
				jsonErrorReport(w, r, "Environment name missing", http.StatusBadRequest)
				return
			}
			if envName != envData["name"].(string) {
				env, err = environment.Get(org, envData["name"].(string))
				if err == nil {
					jsonErrorReport(w, r, "Environment already exists", http.StatusConflict)
					return
				}
				var eerr util.Gerror
				env, eerr = environment.NewFromJSON(org, envData)
				if eerr != nil {
					jsonErrorReport(w, r, eerr.Error(), eerr.Status())
					return
				}
				w.WriteHeader(http.StatusCreated)
				oldenv, olderr := environment.Get(org, envName)
				if olderr == nil {
					oldenv.Delete()
				}
			} else {
				if jsonName == "" {
					envData["name"] = envName
				}
				if err := env.UpdateFromJSON(envData); err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
			}
			if err := env.Save(); err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			if lerr := loginfo.LogEvent(org, opUser, env, "modify"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
		default:
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(&env); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		if delEnv {
			err := env.Delete()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := loginfo.LogEvent(org, opUser, env, "delete"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
		}
		return
	} else if pathArrayLen == 3 {
		envName := vars["name"]
		op := pathArray[2]

		if op == "cookbook_versions" && r.Method != http.MethodPost || op != "cookbook_versions" && r.Method != http.MethodGet {
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}

		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}

		env, err := environment.Get(org, envName)
		if err != nil {
			var errMsg string
			// bleh, stupid errors
			if err.Status() == http.StatusNotFound && (op != "recipes" && op != "cookbooks") {
				errMsg = fmt.Sprintf("environment '%s' not found", envName)
			} else {
				errMsg = err.Error()
			}
			jsonErrorReport(w, r, errMsg, err.Status())
			return
		}

		if f, ferr := org.PermCheck.CheckItemPerm(env, opUser, "read"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}

		switch op {
		case "cookbook_versions":
			/* Chef Server API docs aren't even remotely
			 * right here. What it actually wants is the
			 * usual hash of info for the latest or
			 * constrained version. Weird. */
			cbVer, jerr := parseObjJSON(r.Body)
			if jerr != nil {
				errmsg := jerr.Error()
				if !strings.Contains(errmsg, "Field") {
					errmsg = "invalid JSON"
				} else {
					errmsg = jerr.Error()
				}
				jsonErrorReport(w, r, errmsg, http.StatusBadRequest)
				return
			}

			if _, ok := cbVer["run_list"]; !ok {
				jsonErrorReport(w, r, "POSTed JSON badly formed.", http.StatusMethodNotAllowed)
				return
			}
			deps, derr := cookbook.DependsCookbooks(org, cbVer["run_list"].([]string), env.CookbookVersions)
			if derr != nil {
				switch derr := derr.(type) {
				case *cookbook.DependsError:
					// In 1.0.0-dev, there's a
					// JSONErrorMapReport function in util.
					// Use that when moving this forward
					errMap := make(map[string][]map[string]interface{})
					errMap["error"] = make([]map[string]interface{}, 1)
					errMap["error"][0] = derr.ErrMap()
					w.WriteHeader(http.StatusPreconditionFailed)
					logger.Infof("Cookbook Depends Errors in %s: %+v", env.Name, derr.ErrMap())
					enc := json.NewEncoder(w)
					if jerr := enc.Encode(&errMap); jerr != nil {
						logger.Errorf(jerr.Error())
					}
				default:
					jsonErrorReport(w, r, derr.Error(), http.StatusPreconditionFailed)
				}
				return
			}
			/* Need our own encoding here too. */
			enc := json.NewEncoder(w)
			if err := enc.Encode(&deps); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
			return
		case "cookbooks":
			envResponse = env.AllCookbookHash(numResults)
		case "nodes":
			nodeList, err := node.GetFromEnv(org, envName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			for _, chefNode := range nodeList {
				envResponse[chefNode.Name] = util.ObjURL(chefNode)
			}
		case "recipes":
			envRecipes := env.RecipeList()
			/* And... we have to do our own json response
			 * here. Hmph. */
			/* TODO: make the JSON encoding stuff its own
			 * function. Dunno why I never thought of that
			 * before now for this. */
			enc := json.NewEncoder(w)
			if err := enc.Encode(&envRecipes); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
			return
		default:
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return

		}
	} else if pathArrayLen == 4 {
		envName := vars["name"]
		/* op is either "cookbooks" or "roles", and opName is the name
		 * of the object op refers to. */
		op := pathArray[2]
		opName := vars["op_name"]

		if r.Method != http.MethodGet {
			jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		env, err := environment.Get(org, envName)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		if op == "cookbook_versions" && r.Method != "POST" || op != "cookbook_versions" && r.Method != "GET" {
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}

		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}

		/* Biting the bullet and not redirecting this to
		 * /roles/NAME/environments/NAME. The behavior is exactly the
		 * same, but it makes clients and chef-pedant somewhat unhappy
		 * to not have this way available. */
		if op == "roles" {
			role, err := role.Get(org, opName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			var runList []string
			if envName == "_default" {
				runList = role.RunList
			} else {
				runList = role.EnvRunLists[envName]
			}
			envResponse["run_list"] = runList
		} else if op == "cookbooks" {
			cb, err := cookbook.Get(org, opName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			/* Here and, I think, here only, if num_versions isn't
			 * set it's supposed to return ALL matching versions.
			 * API docs are wrong here. */
			if numResults == "" {
				numResults = "all"
			}
			envResponse[opName] = cb.ConstrainedInfoHash(numResults, env.CookbookVersions[opName])
		} else {
			/* Not an op we know. */
			jsonErrorReport(w, r, "Bad request - too many elements in path", http.StatusBadRequest)
			return
		}
	} else {
		/* Bad number of path elements. */
		jsonErrorReport(w, r, "Bad request - too many elements in path", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&envResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
