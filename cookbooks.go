/* Cookbook functions */

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
	"io/ioutil"
	"net/http"

	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
)

func cookbookHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathArray := splitPath(r.URL.Path)
	cookbookResponse := make(map[string]interface{})

	opUser, oerr := reqctx.CtxReqUser(r.Context())
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

	var force bool
	if f, fok := r.Form["force"]; fok {
		if len(f) > 0 {
			force = true
		}
	}

	pathArrayLen := len(pathArray)

	/* 1 and 2 length path arrays only support GET (or HEAD) */
	if pathArrayLen < 3 && (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		jsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
		return
	} else if pathArrayLen < 3 && opUser.IsValidator() {
		jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
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
	var err error
	if pathArrayLen == 1 || (pathArrayLen == 2 && pathArray[1] == "") {
		if r.Method == http.MethodHead {
			// not, uh, much else to do here
			headResponse(w, r, http.StatusOK)
			return
		}
		/* list all cookbooks */
		cookbookResponse, err = cookbook.CookbookLister(numResults)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if pathArrayLen == 2 {
		/* info about a cookbook and all its versions */
		cookbookName := pathArray[1]

		// Handle HEAD responses here, and avoid wading into all that
		// below
		if r.Method == http.MethodHead {
			// Until something better comes up with these, just send
			// back 200 OK
			if cookbookName == "_latest" || cookbookName == "_recipes" {
				headResponse(w, r, http.StatusOK)
				return
			}
			headChecking(w, r, opUser, cookbookName, cookbook.DoesExist, nilPermCheck)
			return
		}

		/* Undocumented behavior - a cookbook name of _latest gets a
		 * list of the latest versions of all the cookbooks, and _recipe
		 * gets the recipes of the latest cookbooks. */
		var err error
		if cookbookName == "_latest" {
			cookbookResponse, err = cookbook.CookbookLatest()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if cookbookName == "_recipes" {
			rlist, nerr := cookbook.CookbookRecipes()
			if nerr != nil {
				jsonErrorReport(w, r, nerr.Error(), nerr.Status())
				return
			}
			enc := json.NewEncoder(w)
			if err := enc.Encode(&rlist); err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
			return
		} else {
			cb, found, err := cookbook.Get(cookbookName)
			switch {
			case !found:
				jsonErrorReport(w, r, fmt.Sprintf("Cannot find a cookbook named %s", cookbookName), http.StatusNotFound)
				return
			case err != nil:
				logger.Errorf(err.Error())
				jsonErrorReport(w, r, fmt.Sprintf("Cannot get a cookbook named %s", cookbookName), err.Status())
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
		cookbookName := pathArray[1]
		var cookbookVersion string
		var vererr util.Gerror
		opUser, oerr := reqctx.CtxReqUser(r.Context())
		if oerr != nil {
			jsonErrorReport(w, r, oerr.Error(), oerr.Status())
			return
		}
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && pathArray[2] == "_latest" { // might be other special vers
			cookbookVersion = pathArray[2]
		} else {
			cookbookVersion, vererr = util.ValidateAsVersion(pathArray[2])
			if vererr != nil {
				vererr := util.Errorf("Invalid cookbook version '%s'.", pathArray[2])
				jsonErrorReport(w, r, vererr.Error(), vererr.Status())
				return
			}
		}
		switch r.Method {
		case http.MethodHead:
			if opUser.IsValidator() {
				headResponse(w, r, http.StatusForbidden)
				return
			}
			cb, found, err := cookbook.Get(cookbookName)
			switch {
			case !found:
				headResponse(w, r, http.StatusNotFound)
				return
			case err != nil:
				logger.Errorf(err.Error())
				headResponse(w, r, err.Status())
				return
			}
			headChecking(w, r, opUser, cookbookVersion, cb.DoesVersionExist, nilPermCheck)
			return
		case http.MethodDelete, http.MethodGet:
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			//get the cookbook
			cb, found, err := cookbook.Get(cookbookName)
			switch {
			case !found:
				jsonErrorReport(w, r, fmt.Sprintf("Cannot find a cookbook named %s", cookbookName), http.StatusNotFound)
				return
			case err != nil:
				logger.Errorf(err.Error())
				jsonErrorReport(w, r, fmt.Sprintf("Cannot get a cookbook named %s", cookbookName), err.Status())
				return
			}

			//get the specific version of it
			cbv, err := cb.GetVersion(cookbookVersion)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			if cbv == nil {
				jsonErrorReport(
					w, r,
					fmt.Sprintf("Could not find a cookbook named %s with version %s", cookbookName, cookbookVersion),
					http.StatusNotFound)
				return
			}
			if r.Method == http.MethodDelete {
				if !opUser.IsAdmin() {
					jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
					return
				}
				err := cb.DeleteVersion(cookbookVersion)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				if lerr := loginfo.LogEvent(opUser, cbv, "delete"); lerr != nil {
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
				if rs := r.Header.Get("X-Ops-Request-Source"); rs == "web" {
					chkDiv := []string{"definitions", "libraries", "attributes", "providers", "resources", "templates", "root_files", "files"}
					for _, cd := range chkDiv {
						if cookbookResponse[cd] == nil {
							cookbookResponse[cd] = make([]map[string]interface{}, 0)
						}
					}
				}
			}
		case http.MethodPut:
			if !opUser.IsAdmin() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}

			/* First, see if the cookbook already exists, &
			 * if not create it. Second, see if this
			 * specific version of the cookbook exists. If
			 * so, update it, otherwise, create it and set
			 * the latest version as needed. */
			cb, found, err := cookbook.Get(cookbookName)
			if err != nil {
				//there was some kind of unexpected error. report it and quit early
				logger.Errorf(err.Error())
				jsonErrorReport(w, r, fmt.Sprintf("Cannot get a cookbook named %s", cookbookName), err.Status())
				return
			}

			// if we dont find the cookbook, create it
			if !found {
				cb, err = cookbook.New(cookbookName)
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
				if lerr := loginfo.LogEvent(opUser, cb, "create"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
			}

			//get cookbook data from request body and decode it in a proper struct
			jsonBody, jerr := ioutil.ReadAll(r.Body)
			if jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			var newCB cookbook.CookbookVersion
			jerr = json.Unmarshal(jsonBody, &newCB)
			if jerr != nil {
				logger.Errorf("cannot decode json [%s]", string(jsonBody))
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}

			//fetch cookbook from database.
			dbCookbookVersion, err := cb.GetVersion(cookbookVersion)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}

			//if cookbook version doesn't exist, create it
			if dbCookbookVersion == nil {
				var nerr util.Gerror
				dbCookbookVersion, nerr = cb.NewVersion(cookbookVersion, newCB)
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
				if lerr := loginfo.LogEvent(opUser, dbCookbookVersion, "create"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)
				return
			}
			//cookbook version already exist, so we need to update it.
			//validate name
			if newCB.Name != dbCookbookVersion.Name {
				terr := util.Errorf("Field 'name' invalid")
				jsonErrorReport(w, r, terr.Error(), terr.Status())
			}
			err = dbCookbookVersion.UpdateVersion(newCB, force)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			gerr := cb.Save()
			if gerr != nil {
				jsonErrorReport(w, r, gerr.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := loginfo.LogEvent(opUser, dbCookbookVersion, "modify"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
			/* API docs are wrong. The docs claim that this
			 * should have no response body, but in fact it
			 * wants some (not all) of the cookbook version
			 * data. */
			cookbookResponse = dbCookbookVersion.ToJSON(r.Method)
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
