/* Data functions */

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
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/data_bag"
	"github.com/ctdk/goiardi/util"
)

func data_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")

	path_array := SplitPath(r.URL.Path)

	db_response := make(map[string]interface{})
	if len(path_array) == 1 {
		/* Either a list of data bags, or a POST to create a new one */
		switch r.Method {
			case "GET":
				/* The list */
				db_list := data_bag.GetList()
				for _, k := range db_list {
					item_url := fmt.Sprintf("/data/%s", k)
					db_response[k] = util.CustomURL(item_url)
				}
			case "POST":
				db_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				}
				chef_dbag, err := data_bag.Get(db_data["name"].(string))
				if chef_dbag != nil {
					httperr := fmt.Errorf("Data bag %s already exists.", db_data["name"].(string))
					JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
					return
				}
				chef_dbag, err = data_bag.New(db_data["name"].(string))
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				chef_dbag.Save()
				db_response["uri"] = util.ObjURL(chef_dbag)
				w.WriteHeader(http.StatusCreated)
			default:
				JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
				return
		}
	} else { 
		db_name := path_array[1]
		chef_dbag, err := data_bag.Get(db_name)
		if err != nil {
			if r.Method == "POST" && len(path_array) == 2 { 
				/* If we create a data bag item by POSTing to 
				 * /data/NAME and the data bag doesn't exist,
				 * we have to create the data bag to hold the
				 * item despite the API docs implying otherwise.
				 * This fits with previously observed behavior
				 * with knife data bag from file BAG FILE
				 */
				chef_dbag, err = data_bag.New(db_name)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				chef_dbag.Save()
			} else {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}
		}
		if len(path_array) == 2 {
			/* getting list of data bag items and creating data bag
			 * items. */
			switch r.Method {
				case "GET":
					for k, _ := range chef_dbag.DataBagItems {
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
				case "POST":
					raw_data := data_bag.RawDataBagJson(r.Body)
					dbitem, err := chef_dbag.NewDBItem(raw_data)
					if err != nil {
						httperr := fmt.Errorf("Item %s in data bag %s already exists.", raw_data["id"].(string), chef_dbag.Name)
						JsonErrorReport(w, r, httperr.Error(), http.StatusConflict)
						return
					}
					/* The data bag return values are all
					 * kinds of weird. Sometimes it sends
					 * just the raw data, sometimes it sends
					 * the whole object, sometimes a special
					 * snowflake version. Ugh. */
					db_response = dbitem.RawData
					db_response["data_bag"] = dbitem.DataBagName
					db_response["chef_type"] = dbitem.ChefType
					w.WriteHeader(http.StatusCreated)
				default:
					JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
					return
			}
		} else {
			/* getting, editing, and deleting existing data bag items. */
			db_item_name := path_array[2]
			if _, ok := chef_dbag.DataBagItems[db_item_name]; !ok {
				httperr := fmt.Errorf("Item %s in data bag %s does not exist.", db_item_name, chef_dbag.Name)
				JsonErrorReport(w, r, httperr.Error(), http.StatusNotFound)
			}
			switch r.Method {
				case "GET", "DELETE":
					db_response = chef_dbag.DataBagItems[db_item_name].RawData
					if r.Method == "DELETE" {
						err := chef_dbag.DeleteDBItem(db_item_name)
						if err != nil {
							JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
							return
						}
					}
				case "PUT":
					raw_data := data_bag.RawDataBagJson(r.Body)
					dbitem, err := chef_dbag.UpdateDBItem(db_item_name, raw_data)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					/* Another weird data bag item response
					 * which isn't at all unusual. */
					db_response = dbitem.RawData
					db_response["data_bag"] = dbitem.DataBagName
					db_response["chef_type"] = dbitem.ChefType
				default:
					JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
					return
			}
		}
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&db_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}


