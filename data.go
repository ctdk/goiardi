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
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/data_bag"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/log_info"
)

func data_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")

	path_array := SplitPath(r.URL.Path)

	db_response := make(map[string]interface{})
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	if len(path_array) == 1 {
		/* Either a list of data bags, or a POST to create a new one */
		switch r.Method {
			case "GET":
				if opUser.IsValidator() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				/* The list */
				db_list := data_bag.GetList()
				for _, k := range db_list {
					item_url := fmt.Sprintf("/data/%s", k)
					db_response[k] = util.CustomURL(item_url)
				}
			case "POST":
				if !opUser.IsAdmin() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				db_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
					return
				}
				/* check that the name exists */
				switch t := db_data["name"].(type) {
					case string:
						if t == "" {
							JsonErrorReport(w, r, "Field 'name' missing", http.StatusBadRequest)
							return
						}
					default:
						JsonErrorReport(w, r, "Field 'name' missing", http.StatusBadRequest)
						return
				}
				chef_dbag, _ := data_bag.Get(db_data["name"].(string))
				if chef_dbag != nil {
					httperr := fmt.Errorf("Data bag %s already exists.", db_data["name"].(string))
					JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
					return
				}
				chef_dbag, nerr := data_bag.New(db_data["name"].(string))
				if nerr != nil {
					JsonErrorReport(w, r, nerr.Error(), nerr.Status())
					return
				}
				serr := chef_dbag.Save()
				if serr != nil {
					JsonErrorReport(w, r, serr.Error(), http.StatusInternalServerError)
					return
				}
				if lerr := log_info.LogEvent(opUser, chef_dbag, "create"); lerr != nil {
					JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
					return
				}
				db_response["uri"] = util.ObjURL(chef_dbag)
				w.WriteHeader(http.StatusCreated)
			default:
				/* The chef-pedant spec wants this response for
				 * some reason. Mix it up, I guess. */
				w.Header().Set("Allow", "GET, POST")
				JsonErrorReport(w, r, "GET, POST", http.StatusMethodNotAllowed)
				return
		}
	} else { 
		db_name := path_array[1]

		/* chef-pedant is unhappy about not reporting the HTTP status
		 * as 404 by fetching the data bag before we see if the method
		 * is allowed, so do a quick check for that here. */
		if (len(path_array) == 2  && r.Method == "PUT") || (len(path_array) == 3 && r.Method == "POST"){
			var allowed string
			if len(path_array) == 2 {
				allowed = "GET, POST, DELETE"
			} else {
				allowed = "GET, PUT, DELETE"
			}
			w.Header().Set("Allow", allowed)
			JsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if opUser.IsValidator() || (!opUser.IsAdmin() && r.Method != "GET") {
			JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		chef_dbag, err := data_bag.Get(db_name)
		if err != nil {
			var err_msg string
			status := err.Status()
			if r.Method == "POST" {
				/* Posts get a special snowflake message */
				err_msg = fmt.Sprintf("No data bag '%s' could be found. Please create this data bag before adding items to it.", db_name)
			} else {
				if len(path_array) == 3 {
					/* This is nuts. */
					if r.Method == "DELETE" {
						err_msg = fmt.Sprintf("Cannot load data bag %s item %s", db_name, path_array[2])
					} else {
						err_msg = fmt.Sprintf("Cannot load data bag item %s for data bag %s", path_array[2], db_name)
					}
				} else {
					err_msg = err.Error()
				}
			}
			JsonErrorReport(w, r, err_msg, status)
			return
		}
		if len(path_array) == 2 {
			/* getting list of data bag items and creating data bag
			 * items. */
			switch r.Method {
				case "GET":
					
					for _, k := range chef_dbag.ListDBItems() {
						db_response[k] = util.CustomObjURL(chef_dbag, k)
					}
				case "DELETE":
					/* The chef API docs don't say anything
					 * about this existing, but it does,
					 * and without it you can't delete data
					 * bags at all. */
					db_response["chef_type"] = "data_bag"
					db_response["json_class"] = "Chef::DataBag"
					db_response["name"] = chef_dbag.Name
					err := chef_dbag.Delete()
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					if lerr := log_info.LogEvent(opUser, chef_dbag, "delete"); lerr != nil {
						JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
						return
					}
				case "POST":
					raw_data := data_bag.RawDataBagJson(r.Body)
					dbitem, nerr := chef_dbag.NewDBItem(raw_data)
					if nerr != nil {
						JsonErrorReport(w, r, nerr.Error(), nerr.Status())
						return
					}
					if lerr := log_info.LogEvent(opUser, dbitem, "create"); lerr != nil {
						JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
						return
					}
					
					/* The data bag return values are all
					 * kinds of weird. Sometimes it sends
					 * just the raw data, sometimes it sends
					 * the whole object, sometimes a special
					 * snowflake version. Ugh. Have to loop
					 * through to avoid updating the pointer
					 * in the cache by just assigning
					 * dbitem.RawData to db_response. Urk.
					 */
					for k, v := range dbitem.RawData {
						db_response[k] = v
					}
					db_response["data_bag"] = dbitem.DataBagName
					db_response["chef_type"] = dbitem.ChefType
					w.WriteHeader(http.StatusCreated)
				default:
					w.Header().Set("Allow", "GET, DELETE, POST")
					JsonErrorReport(w, r, "GET, DELETE, POST", http.StatusMethodNotAllowed)
					return
			}
		} else {
			/* getting, editing, and deleting existing data bag items. */
			db_item_name := path_array[2]
			if _, err := chef_dbag.GetDBItem(db_item_name); err != nil {
				var httperr string
				if r.Method != "DELETE" {
					httperr = fmt.Sprintf("Cannot load data bag item %s for data bag %s", db_item_name, chef_dbag.Name)
				} else {
					httperr = fmt.Sprintf("Cannot load data bag %s item %s", chef_dbag.Name, db_item_name)
				}
				JsonErrorReport(w, r, httperr, http.StatusNotFound)
				return
			}
			switch r.Method {
				case "GET":
					dbi, err := chef_dbag.GetDBItem(db_item_name)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					db_response = dbi.RawData
				case "DELETE":
					dbi, err := chef_dbag.GetDBItem(db_item_name)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					/* Gotta short circuit this */
					enc := json.NewEncoder(w)
					if err := enc.Encode(&dbi); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					err = chef_dbag.DeleteDBItem(db_item_name)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					if lerr := log_info.LogEvent(opUser, dbi, "delete"); lerr != nil {
						JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
						return
					}
					return
				case "PUT":
					raw_data := data_bag.RawDataBagJson(r.Body)
					if raw_id, ok := raw_data["id"]; ok {
						switch raw_id := raw_id.(type) {
							case string:
								if raw_id != db_item_name {
									JsonErrorReport(w, r, "DataBagItem name mismatch.", http.StatusBadRequest)
									return
								}
							default:
								JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
								return
						}
					}
					dbitem, err := chef_dbag.UpdateDBItem(db_item_name, raw_data)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					if lerr := log_info.LogEvent(opUser, dbitem, "modify"); lerr != nil {
						JsonErrorReport(w, r, lerr.Error(), http.StatusInternalServerError)
						return
					}
					/* Another weird data bag item response
					 * which isn't at all unusual. */
					for k, v := range dbitem.RawData {
						db_response[k] = v
					}
					db_response["data_bag"] = dbitem.DataBagName
					db_response["chef_type"] = dbitem.ChefType
					db_response["id"] = db_item_name
				default:
					w.Header().Set("Allow", "GET, DELETE, PUT")
					JsonErrorReport(w, r, "GET, DELETE, PUT", http.StatusMethodNotAllowed)
					return
			}
		}
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&db_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}


