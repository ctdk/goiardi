/* Cookbook functions */

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
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func cookbookHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathArray := splitPath(r.URL.Path)[2:]
	cookbookResponse := make(map[string]interface{})

	vars := mux.Vars(r)
	org, orgerr := organization.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}

	opUser, oerr := actor.GetReqUser(org, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

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
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
	}
	force := ""
	if f, fok := r.Form["force"]; fok {
		if len(f) > 0 {
			force = f[0]
		}
	}

	pathArrayLen := len(pathArray)

	/* 1 and 2 length path arrays only support GET */
	if pathArrayLen < 3 && r.Method != "GET" {
		jsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
		return
	} else if pathArrayLen < 3 && opUser.IsValidator() {
		jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
		return
	}

	// check container perms
	containerACL, conerr := acl.Get(org, "containers", "cookbooks")
	if conerr != nil {
		jsonErrorReport(w, r, conerr.Error(), conerr.Status())
		return
	}

	/* chef-pedant is happier when checking if a validator can do something
	 * surprisingly late in the game. It wants the perm checks to be
	 * checked after the method for the end point is checked out as
	 * something it's going to handle, so, for instance, issuing a DELETE
	 * against an endpoint where that isn't allowed should respond with a
	 * 405, rather than a 403, which also makes sense in areas where
	 * validators shouldn't be able to do anything. *shrugs*
	 */

	if pathArrayLen == 1 || (pathArrayLen == 2 && pathArray[1] == "") {
		if f, ferr := containerACL.CheckPerm("read", opUser); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		/* list all cookbooks */
		cookbookResponse = cookbook.CookbookLister(org, numResults)
	} else if pathArrayLen == 2 {
		/* info about a cookbook and all its versions */
		cookbookName := vars["name"]
		/* Undocumented behavior - a cookbook name of _latest gets a
		 * list of the latest versions of all the cookbooks, and _recipe
		 * gets the recipes of the latest cookbooks. */
		if cookbookName == "_latest" || cookbookName == "_recipes" {
			if f, ferr := containerACL.CheckPerm("read", opUser); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
				return
			}
			if cookbookName == "_latest" {
				cookbookResponse = cookbook.CookbookLatest(org)
			} else if cookbookName == "_recipes" {
				rlist, nerr := cookbook.CookbookRecipes(org)
				if nerr != nil {
					jsonErrorReport(w, r, nerr.Error(), nerr.Status())
					return
				}
				enc := json.NewEncoder(w)
				if err := enc.Encode(&rlist); err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		} else {
			cb, err := cookbook.Get(org, cookbookName)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			cbACL, cberr := acl.GetItemACL(org, cb)
			if cberr != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			if f, ferr := cbACL.CheckPerm("read", opUser); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
				return
			}
			/* Strange thing here. The API docs say if num_versions
			 * is not specified to return one cookbook, yet the
			 * spec indicates that if it's not set that all
			 * cookbooks should be returned. Most places *except
			 * here* that's the case, so it can't be changed in
			 * infoHashBase. Explicitly set numResults to all
			 * here. */
			if numResults == "" {
				numResults = "all"
			}
			cookbookResponse[cookbookName] = cb.InfoHash(numResults)
		}
	} else if pathArrayLen == 3 {
		/* get information about or manipulate a specific cookbook
		 * version */
		cookbookName := vars["name"]
		var cookbookVersion string
		var vererr util.Gerror
		vbase := vars["version"]
		if r.Method == "GET" && vbase == "_latest" { // might be other special vers
			cookbookVersion = vbase
		} else {
			cookbookVersion, vererr = util.ValidateAsVersion(vbase)
			if vererr != nil {
				vererr := util.Errorf("Invalid cookbook version '%s'.", vbase)
				jsonErrorReport(w, r, vererr.Error(), vererr.Status())
				return
			}
		}
		switch r.Method {
		case "DELETE", "GET":
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			cb, err := cookbook.Get(org, cookbookName)
			if err != nil {
				if err.Status() == http.StatusNotFound {
					msg := fmt.Sprintf("Cannot find a cookbook named %s with version %s", cookbookName, cookbookVersion)
					jsonErrorReport(w, r, msg, err.Status())
				} else {
					jsonErrorReport(w, r, err.Error(), err.Status())
				}
				return
			}
			cbv, err := cb.GetVersion(cookbookVersion)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			cbACL, cberr := acl.GetItemACL(org, cb)
			if cberr != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			if r.Method == "DELETE" {
				// do we need to track perms beyond the
				// container ones?
				if f, err := cbACL.CheckPerm("delete", opUser); err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else if !f {
					jsonErrorReport(w, r, "missing delete permission", http.StatusForbidden)
					return
				}
				
				err := cb.DeleteVersion(cookbookVersion)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, cbv, "delete"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
				/* If all versions are gone, remove the
				 * cookbook - seems to be the desired
				 * behavior. */
				if cb.NumVersions() == 0 {
					if cerr := cb.Delete(); cerr != nil {
						jsonErrorReport(w, r, cerr.Error(), http.StatusInternalServerError)
						return
					}
				}
			} else {
				/* Special JSON rendition of the
				 * cookbook with some but not all of
				 * the fields. */
				cookbookResponse = cbv.ToJSON(r.Method)
				/* Sometimes, but not always, chef needs
				 * empty slices of maps for these
				 * values. Arrrgh. */
				/* Doing it this way is absolutely
				 * insane. However, webui really wants
				 * this information, while chef-pedant
				 * absolutely does NOT want it there.
				 * knife seems happy without it as well.
				 * Until such time that this gets
				 * resolved in a non-crazy manner, for
				 * this only send that info back if it's
				 * a webui request. */
				if f, err := cbACL.CheckPerm("read", opUser); err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else if !f {
					jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
					return
				}
				if rs := r.Header.Get("X-Ops-Request-Source"); rs == "web" {
					chkDiv := []string{"definitions", "libraries", "attributes", "providers", "resources", "templates", "root_files", "files"}
					for _, cd := range chkDiv {
						if cookbookResponse[cd] == nil {
							cookbookResponse[cd] = make([]map[string]interface{}, 0)
						}
					}
				}
			}
		case "PUT":
			cbvData, jerr := parseObjJSON(r.Body)
			if jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			/* First, see if the cookbook already exists, &
			 * if not create it. Second, see if this
			 * specific version of the cookbook exists. If
			 * so, update it, otherwise, create it and set
			 * the latest version as needed. */
			cb, err := cookbook.Get(org, cookbookName)
			if err != nil {
				if f, err := containerACL.CheckPerm("create", opUser); err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else if !f {
					jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
					return
				}
				cb, err = cookbook.New(org, cookbookName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				/* save it so we get the id with mysql
				 * for creating versions & such */
				serr := cb.Save()
				if serr != nil {
					jsonErrorReport(w, r, serr.Error(), http.StatusInternalServerError)
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, cb, "create"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				cbACL, cberr := acl.GetItemACL(org, cb)
				if cberr != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				if f, err := cbACL.CheckPerm("update", opUser); err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				} else if !f {
					jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
					return
				}
			}
			cbv, err := cb.GetVersion(cookbookVersion)

			/* Does the cookbook_name in the URL and what's
			 * in the body match? */
			switch t := cbvData["cookbook_name"].(type) {
			case string:
				/* Only send this particular
				 * error if the cookbook version
				 * hasn't been created yet.
				 * Instead we want a slightly
				 * different version later. */
				if t != cookbookName && cbv == nil {
					terr := util.Errorf("Field 'name' invalid")
					jsonErrorReport(w, r, terr.Error(), terr.Status())
					return
				}
			default:
				// rather unlikely, I think, to
				// be able to get here past the
				// cookbook get. Punk out and
				// don't do anything

			}
			if err != nil {
				var nerr util.Gerror
				cbv, nerr = cb.NewVersion(cookbookVersion, cbvData)
				if nerr != nil {
					// If the new version failed to
					// take, and there aren't any
					// other versions of the cookbook
					// it needs to be deleted.
					if cb.NumVersions() == 0 {
						cb.Delete()
					}
					jsonErrorReport(w, r, nerr.Error(), nerr.Status())
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, cbv, "create"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)
			} else {
				err := cbv.UpdateVersion(cbvData, force)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				gerr := cb.Save()
				if gerr != nil {
					jsonErrorReport(w, r, gerr.Error(), http.StatusInternalServerError)
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, cbv, "modify"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
			}
			/* API docs are wrong. The docs claim that this
			 * should have no response body, but in fact it
			 * wants some (not all) of the cookbook version
			 * data. */
			cookbookResponse = cbv.ToJSON(r.Method)
		default:
			jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
			return
		}
	} else {
		/* Say what? Bad request. */
		jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&cookbookResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
