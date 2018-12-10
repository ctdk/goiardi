/* Data functions */

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
	//"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func dataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	pathArray := splitPath(r.URL.Path)[2:]
	pathArrayLen := len(pathArray)

	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	dbResponse := make(map[string]interface{})

	if pathArrayLen == 1 {
		if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "data", "read"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		/* Either a list of data bags, or a POST to create a new one */
		switch r.Method {
		case http.MethodGet:
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			/* The list */
			dbList := databag.GetList(org)
			for _, k := range dbList {
				dbResponse[k] = util.CustomURL(util.JoinStr("/organizations/", org.Name, "/data/", k))
			}
		case http.MethodHead:
			if opUser.IsValidator() {
				headResponse(w, r, http.StatusForbidden)
				return
			}
			headDefaultResponse(w, r)
			return
		case http.MethodPost:
			if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "data", "create"); ferr != nil {
				jsonErrorReport(w, r, ferr.Error(), ferr.Status())
				return
			} else if !f {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			dbData, jerr := parseObjJSON(r.Body)
			if jerr != nil {
				jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			/* check that the name exists */
			switch t := dbData["name"].(type) {
			case string:
				if t == "" {
					jsonErrorReport(w, r, "Field 'name' missing", http.StatusBadRequest)
					return
				}
			default:
				jsonErrorReport(w, r, "Field 'name' missing", http.StatusBadRequest)
				return
			}
			chefDbag, _ := databag.Get(org, dbData["name"].(string))
			if chefDbag != nil {
				httperr := fmt.Errorf("Data bag %s already exists.", dbData["name"].(string))
				jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
				return
			}
			chefDbag, nerr := databag.New(org, dbData["name"].(string))
			if nerr != nil {
				jsonErrorReport(w, r, nerr.Error(), nerr.Status())
				return
			}
			serr := chefDbag.Save()
			if serr != nil {
				jsonErrorReport(w, r, serr.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := loginfo.LogEvent(org, opUser, chefDbag, "create"); lerr != nil {
				jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
				return
			}
			dbResponse["uri"] = util.ObjURL(chefDbag)
			w.WriteHeader(http.StatusCreated)
		default:
			/* The chef-pedant spec wants this response for
			 * some reason. Mix it up, I guess. */
			w.Header().Set("Allow", "GET, POST")
			jsonErrorReport(w, r, "GET, POST", http.StatusMethodNotAllowed)
			return
		}
	} else {
		dbName := vars["name"]

		/*
		 * HEAD response note:
		 * at this time, chef-pedant will flip if these responses start
		 * changing to allow HEAD (or at least say it's OK). It'll be
		 * inaccurate with reporting what methods are allowable at least
		 * for a little while.
		 */

		/* chef-pedant is unhappy about not reporting the HTTP status
		 * as 404 by fetching the data bag before we see if the method
		 * is allowed, so do a quick check for that here. */

		if (pathArrayLen == 2 && r.Method == http.MethodPut) || (pathArrayLen == 3 && r.Method == http.MethodPost) {
			var allowed string
			if pathArrayLen == 2 {
				allowed = "GET, POST, DELETE"
			} else {
				allowed = "GET, PUT, DELETE"
			}
			w.Header().Set("Allow", allowed)
			jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if opUser.IsValidator() {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}

		// This is what was happening in the auth-1.3 branch taken from
		// the 0.11.3 master branch for HEAD responses. Since the
		// IsAdmin() function isn't really productive anymore, this is
		// going to need refactoring to fit with the 1.0.0 permissions.
		// Commenting handling HEAD out for data bags for now until it
		// gets sorted.
		/**************************************************************
		if opUser.IsValidator() || (!opUser.IsAdmin() && (r.Method != http.MethodGet && r.Method != http.MethodHead)) {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}

		// Do HEAD responses here, before starting to fetch full data
		// bags and the like.
		if r.Method == http.MethodHead {
			permCheck := func(r *http.Request, dbName string, opUser actor.Actor) util.Gerror {
				if opUser.IsValidator() {
					return headForbidden()
				}
				return nil
			}
			if len(pathArray) == 2 {

				headChecking(w, r, opUser, org, dbName, databag.DoesExist, permCheck)
			} else {
				dbItemName := pathArray[2]
				chefDbag, err := databag.Get(dbName)
				if err != nil {
					headResponse(w, r, err.Status())
					return
				}
				headChecking(w, r, opUser, org, dbItemName, chefDbag.DoesItemExist, permCheck)
				return
			}
			return
		}
		**************************************************************/
		chefDbag, err := databag.Get(org, dbName)
		if err != nil {
			var errMsg string
			status := err.Status()
			if r.Method == http.MethodPost {
				/* Posts get a special snowflake message */
				errMsg = fmt.Sprintf("No data bag '%s' could be found. Please create this data bag before adding items to it.", dbName)
			} else {
				if pathArrayLen == 3 {
					/* This is nuts. */
					if r.Method == http.MethodDelete {
						errMsg = fmt.Sprintf("Cannot load data bag %s item %s", dbName, vars["item"])
					} else {
						errMsg = fmt.Sprintf("Cannot load data bag item %s for data bag %s", vars["item"], dbName)
					}
				} else {
					errMsg = err.Error()
				}
			}
			jsonErrorReport(w, r, errMsg, status)
			return
		}

		var permstr string
		switch r.Method {
		case "GET":
			permstr = "read"
		case "DELETE":
			permstr = "delete"
		case "PUT":
			permstr = "update"
		case "POST":
			permstr = "create"
		default:
			if pathArrayLen == 2 {
				w.Header().Set("Allow", "GET, DELETE, POST")
				jsonErrorReport(w, r, "GET, DELETE, POST", http.StatusMethodNotAllowed)
				return
			} else {
				w.Header().Set("Allow", "GET, DELETE, PUT")
				jsonErrorReport(w, r, "GET, DELETE, PUT", http.StatusMethodNotAllowed)
				return
			}
		}

		if f, ferr := org.PermCheck.CheckItemPerm(chefDbag, opUser, permstr); ferr != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}

		if pathArrayLen == 2 {
			/* getting list of data bag items and creating data bag
			 * items. */
			switch r.Method {
			case http.MethodGet:
				for _, k := range chefDbag.ListDBItems() {
					dbResponse[k] = util.CustomObjURL(chefDbag, k)
				}
			case http.MethodDelete:
				/* The chef API docs don't say anything
				 * about this existing, but it does,
				 * and without it you can't delete data
				 * bags at all. */
				dbResponse["chef_type"] = "data_bag"
				dbResponse["json_class"] = "Chef::DataBag"
				dbResponse["name"] = chefDbag.Name
				err := chefDbag.Delete()
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, chefDbag, "delete"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
			case http.MethodPost:
				rawData := databag.RawDataBagJSON(r.Body)
				dbitem, nerr := chefDbag.NewDBItem(rawData)
				if nerr != nil {
					jsonErrorReport(w, r, nerr.Error(), nerr.Status())
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, dbitem, "create"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}

				/* The data bag return values are all
				 * kinds of weird. Sometimes it sends
				 * just the raw data, sometimes it sends
				 * the whole object, sometimes a special
				 * snowflake version. Ugh. Have to loop
				 * through to avoid updating the pointer
				 * in the cache by just assigning
				 * dbitem.RawData to dbResponse. Urk.
				 */
				for k, v := range dbitem.RawData {
					dbResponse[k] = v
				}
				dbResponse["data_bag"] = dbitem.DataBagName
				dbResponse["chef_type"] = dbitem.ChefType
				w.WriteHeader(http.StatusCreated)
			default:
				w.Header().Set("Allow", "GET, DELETE, POST")
				jsonErrorReport(w, r, "GET, DELETE, POST", http.StatusMethodNotAllowed)
				return
			}
		} else {
			/* getting, editing, and deleting existing data bag items. */
			dbItemName := vars["item"]
			if _, err := chefDbag.GetDBItem(dbItemName); err != nil {
				var httperr string
				if r.Method != http.MethodDelete {
					httperr = fmt.Sprintf("Cannot load data bag item %s for data bag %s", dbItemName, chefDbag.Name)
				} else {
					httperr = fmt.Sprintf("Cannot load data bag %s item %s", chefDbag.Name, dbItemName)
				}
				jsonErrorReport(w, r, httperr, http.StatusNotFound)
				return
			}
			switch r.Method {
			case http.MethodGet:
				dbi, err := chefDbag.GetDBItem(dbItemName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				dbResponse = dbi.RawData
			case http.MethodDelete:
				dbi, err := chefDbag.GetDBItem(dbItemName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				/* Gotta short circuit this */
				enc := json.NewEncoder(w)
				if err := enc.Encode(&dbi); err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				err = chefDbag.DeleteDBItem(dbItemName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, dbi, "delete"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
				return
			case http.MethodPut:
				rawData := databag.RawDataBagJSON(r.Body)
				if rawID, ok := rawData["id"]; ok {
					switch rawID := rawID.(type) {
					case string:
						if rawID != dbItemName {
							jsonErrorReport(w, r, "DataBagItem name mismatch.", http.StatusBadRequest)
							return
						}
					default:
						jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
						return
					}
				}
				dbitem, err := chefDbag.UpdateDBItem(dbItemName, rawData)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				if lerr := loginfo.LogEvent(org, opUser, dbitem, "modify"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
				/* Another weird data bag item response
				 * which isn't at all unusual. */
				for k, v := range dbitem.RawData {
					dbResponse[k] = v
				}
				dbResponse["data_bag"] = dbitem.DataBagName
				dbResponse["chef_type"] = dbitem.ChefType
				dbResponse["id"] = dbItemName
			default:
				w.Header().Set("Allow", "GET, DELETE, PUT")
				jsonErrorReport(w, r, "GET, DELETE, PUT", http.StatusMethodNotAllowed)
				return
			}
		}
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&dbResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
