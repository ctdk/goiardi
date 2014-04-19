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
	"net/http"
	"encoding/json"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/util"
	"fmt"
	"sort"
	"github.com/ctdk/goiardi/actor"
)

func cookbook_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	path_array := SplitPath(r.URL.Path)
	cookbook_response := make(map[string]interface{})

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

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
			JsonErrorReport(w, r, err.Error(), err.Status())
			return
		}
	}
	force := ""
	if f, fok := r.Form["force"]; fok {
		if len(f) > 0 {
			force = f[0]
		}
	}
	
	path_array_len := len(path_array)

	/* 1 and 2 length path arrays only support GET */
	if path_array_len < 3 && r.Method != "GET" {
		JsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
		return
	} else if path_array_len < 3 && opUser.IsValidator() {
		JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
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

	if path_array_len == 1 {
		/* list all cookbooks */
		for _, cb := range cookbook.AllCookbooks() {
			cookbook_response[cb.Name] = cb.InfoHash(num_results)
		}
	} else if path_array_len == 2 {
		/* info about a cookbook and all its versions */
		cookbook_name := path_array[1]
		/* Undocumented behavior - a cookbook name of _latest gets a 
		 * list of the latest versions of all the cookbooks, and _recipe
		 * gets the recipes of the latest cookbooks. */
		rlist := make([]string, 0)
		if cookbook_name == "_latest" || cookbook_name == "_recipes" {
			for _, cb := range cookbook.AllCookbooks() {
				if cookbook_name == "_latest" {
					cookbook_response[cb.Name] = util.CustomObjURL(cb, cb.LatestVersion().Version)
				} else {
					/* Damn it, this sends back an array of
					 * all the recipes. Fill it in, and send
					 * back the JSON ourselves. */
					rlist_tmp, _ := cb.LatestVersion().RecipeList()
					rlist = append(rlist, rlist_tmp...)
				}
				sort.Strings(rlist)
			}
			if cookbook_name == "_recipes" {
				enc := json.NewEncoder(w)
				if err := enc.Encode(&rlist); err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		} else {
			cb, err := cookbook.Get(cookbook_name)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
			/* Strange thing here. The API docs say if num_versions
			 * is not specified to return one cookbook, yet the 
			 * spec indicates that if it's not set that all 
			 * cookbooks should be returned. Most places *except 
			 * here* that's the case, so it can't be changed in 
			 * infoHashBase. Explicitly set num_results to all 
			 * here. */
			if num_results == "" {
				num_results = "all"
			}
			cookbook_response[cookbook_name] = cb.InfoHash(num_results)
		}
	} else if path_array_len == 3 {
		/* get information about or manipulate a specific cookbook
		 * version */
		cookbook_name := path_array[1]
		var cookbook_version string
		var vererr util.Gerror
		opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
		if oerr != nil {
			JsonErrorReport(w, r, oerr.Error(), oerr.Status())
			return
		}
		if r.Method == "GET" && path_array[2] == "_latest" {  // might be other special vers
			cookbook_version = path_array[2]
		} else {
			cookbook_version, vererr = util.ValidateAsVersion(path_array[2]);
			if vererr != nil {
				vererr := util.Errorf("Invalid cookbook version '%s'.", path_array[2])
				JsonErrorReport(w, r, vererr.Error(), vererr.Status())
				return
			}
		}
		switch r.Method {
			case "DELETE", "GET":
				if opUser.IsValidator() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				cb, err := cookbook.Get(cookbook_name)
				if err != nil {
					if err.Status() == http.StatusNotFound {
						msg := fmt.Sprintf("Cannot find a cookbook named %s with version %s", cookbook_name, cookbook_version)
						JsonErrorReport(w, r, msg, err.Status())
					} else {
						JsonErrorReport(w, r, err.Error(), err.Status())
					}
					return
				}
				cb_ver, err := cb.GetVersion(cookbook_version)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
					return
				}
				if r.Method == "DELETE" {
					if !opUser.IsAdmin(){
						JsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
						return
					}
					err := cb.DeleteVersion(cookbook_version)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					/* If all versions are gone, remove the
					 * cookbook - seems to be the desired
					 * behavior. */
					if cb.NumVersions() == 0 {
						if cerr := cb.Delete(); cerr != nil {
							JsonErrorReport(w, r, cerr.Error(), http.StatusInternalServerError)
							return
						}
					}
				} else {
					/* Special JSON rendition of the 
					 * cookbook with some but not all of
					 * the fields. */
					cookbook_response = cb_ver.ToJson(r.Method)
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
					if rs := r.Header.Get("X-Ops-Request-Source"); rs == "web" {
						chkDiv := []string{ "definitions", "libraries", "attributes", "providers", "resources", "templates", "root_files", "files" }
						for _, cd := range chkDiv {
							if cookbook_response[cd] == nil {
								cookbook_response[cd] = make([]map[string]interface{}, 0)
							}
						}
					}
				}
			case "PUT":
				if !opUser.IsAdmin() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				cbv_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
					return
				}
				/* First, see if the cookbook already exists, &
				 * if not create it. Second, see if this 
				 * specific version of the cookbook exists. If
				 * so, update it, otherwise, create it and set
				 * the latest version as needed. */
				cb, err := cookbook.Get(cookbook_name)
				if err != nil {
					cb, err = cookbook.New(cookbook_name)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					/* save it so we get the id with mysql
					 * for createing versions & such */
					serr := cb.Save()
					if serr != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				cbv, err := cb.GetVersion(cookbook_version)
				
				/* Does the cookbook_name in the URL and what's
				 * in the body match? */
				switch t := cbv_data["cookbook_name"].(type) {
					case string:
						/* Only send this particular
						 * error if the cookbook version
						 * hasn't been created yet.
						 * Instead we want a slightly
						 * different version later. */
						if t != cookbook_name && cbv == nil {
							terr := util.Errorf("Field 'name' invalid")
							JsonErrorReport(w, r, terr.Error(), terr.Status())
							return 
						}
					default:
						// rather unlikely, I think, to
						// be able to get here past the
						// cookbook get. Punk out and
						// don't do anything
						;
				}
				if err != nil {
					var nerr util.Gerror
					cbv, nerr = cb.NewVersion(cookbook_version, cbv_data)
					if nerr != nil {
						// If the new version failed to
						// take, and there aren't any
						// other versions of the cookbook
						// it needs to be deleted.
						if cb.NumVersions() == 0 {
							cb.Delete()
						}
						JsonErrorReport(w, r, nerr.Error(), nerr.Status())
						return
					}
					w.WriteHeader(http.StatusCreated)
				} else {
					err := cbv.UpdateVersion(cbv_data, force)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), err.Status())
						return
					} else {
						err := cb.Save()
						if err != nil {
							JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
							return
						}
					}
				}
				/* API docs are wrong. The docs claim that this
				 * should have no response body, but in fact it
				 * wants some (not all) of the cookbook version
				 * data. */
				cookbook_response = cbv.ToJson(r.Method)
			default:
				JsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
				return
		}
	} else {
		/* Say what? Bad request. */
		JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&cookbook_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
