/* Data functions */

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
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func dataHandler(org *organization.Organization, w http.ResponseWriter, r *http.Request) {
	_ = org
	w.Header().Set("Content-Type", "application/json")

	pathArray := splitPath(r.URL.Path)[2:]

	dbResponse := make(map[string]interface{})
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	if len(pathArray) == 1 {
		/* Either a list of data bags, or a POST to create a new one */
		switch r.Method {
		case "GET":
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			/* The list */
			dbList := databag.GetList()
			for _, k := range dbList {
				dbResponse[k] = util.CustomURL(fmt.Sprintf("/data/%s", k))
			}
		case "POST":
			if !opUser.IsAdmin() {
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
			chefDbag, _ := databag.Get(dbData["name"].(string))
			if chefDbag != nil {
				httperr := fmt.Errorf("Data bag %s already exists.", dbData["name"].(string))
				jsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
				return
			}
			chefDbag, nerr := databag.New(dbData["name"].(string))
			if nerr != nil {
				jsonErrorReport(w, r, nerr.Error(), nerr.Status())
				return
			}
			serr := chefDbag.Save()
			if serr != nil {
				jsonErrorReport(w, r, serr.Error(), http.StatusInternalServerError)
				return
			}
			if lerr := loginfo.LogEvent(opUser, chefDbag, "create"); lerr != nil {
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
		dbName := pathArray[1]

		/* chef-pedant is unhappy about not reporting the HTTP status
		 * as 404 by fetching the data bag before we see if the method
		 * is allowed, so do a quick check for that here. */
		if (len(pathArray) == 2 && r.Method == "PUT") || (len(pathArray) == 3 && r.Method == "POST") {
			var allowed string
			if len(pathArray) == 2 {
				allowed = "GET, POST, DELETE"
			} else {
				allowed = "GET, PUT, DELETE"
			}
			w.Header().Set("Allow", allowed)
			jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if opUser.IsValidator() || (!opUser.IsAdmin() && r.Method != "GET") {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		chefDbag, err := databag.Get(dbName)
		if err != nil {
			var errMsg string
			status := err.Status()
			if r.Method == "POST" {
				/* Posts get a special snowflake message */
				errMsg = fmt.Sprintf("No data bag '%s' could be found. Please create this data bag before adding items to it.", dbName)
			} else {
				if len(pathArray) == 3 {
					/* This is nuts. */
					if r.Method == "DELETE" {
						errMsg = fmt.Sprintf("Cannot load data bag %s item %s", dbName, pathArray[2])
					} else {
						errMsg = fmt.Sprintf("Cannot load data bag item %s for data bag %s", pathArray[2], dbName)
					}
				} else {
					errMsg = err.Error()
				}
			}
			jsonErrorReport(w, r, errMsg, status)
			return
		}
		if len(pathArray) == 2 {
			/* getting list of data bag items and creating data bag
			 * items. */
			switch r.Method {
			case "GET":

				for _, k := range chefDbag.ListDBItems() {
					dbResponse[k] = util.CustomObjURL(chefDbag, k)
				}
			case "DELETE":
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
				if lerr := loginfo.LogEvent(opUser, chefDbag, "delete"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
			case "POST":
				rawData := databag.RawDataBagJSON(r.Body)
				dbitem, nerr := chefDbag.NewDBItem(rawData)
				if nerr != nil {
					jsonErrorReport(w, r, nerr.Error(), nerr.Status())
					return
				}
				if lerr := loginfo.LogEvent(opUser, dbitem, "create"); lerr != nil {
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
			dbItemName := pathArray[2]
			if _, err := chefDbag.GetDBItem(dbItemName); err != nil {
				var httperr string
				if r.Method != "DELETE" {
					httperr = fmt.Sprintf("Cannot load data bag item %s for data bag %s", dbItemName, chefDbag.Name)
				} else {
					httperr = fmt.Sprintf("Cannot load data bag %s item %s", chefDbag.Name, dbItemName)
				}
				jsonErrorReport(w, r, httperr, http.StatusNotFound)
				return
			}
			switch r.Method {
			case "GET":
				dbi, err := chefDbag.GetDBItem(dbItemName)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				dbResponse = dbi.RawData
			case "DELETE":
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
				if lerr := loginfo.LogEvent(opUser, dbi, "delete"); lerr != nil {
					jsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
				return
			case "PUT":
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
				if lerr := loginfo.LogEvent(opUser, dbitem, "modify"); lerr != nil {
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
