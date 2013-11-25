/* Cookbook functions */

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
	"log"
	"encoding/json"
	"github.com/ctdk/goiardi/cookbook"
)

func cookbook_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	path_array := SplitPath(r.URL.Path)
	cookbook_response := make(map[string]interface{})

	num_results := r.FormValue("num_versions")
	
	path_array_len := len(path_array)

	/* 1 and 2 length path arrays only support GET */
	if path_array_len < 3 && r.Method != "GET" {
		JsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
		return
	}

	if path_array_len == 1 {
		/* list all cookbooks */
		cookbook_list := cookbook.GetList()
		for _, c := range cookbook_list {
			cb, err := cookbook.Get(c)
			if err != nil {
				log.Printf("Curious. Cookbook %s was in the cookbook list, but wasn't found when fetched. Continuing.", c)
				continue
			}
			cookbook_response[cb.Name] = cb.InfoHash(num_results)
		}
	} else if path_array_len == 2 {
		/* info about a cookbook and all its versions */
		cookbook_name := path_array[1]
		cb, err := cookbook.Get(cookbook_name)
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		cookbook_response[cookbook_name] = cb.InfoHash(num_results)
	} else if path_array_len == 3 {
		/* get information about or manipulate a specific cookbook
		 * version */
		cookbook_name := path_array[1]
		cookbook_version := path_array[2]
		switch r.Method {
			case "DELETE", "GET":
				cb, err := cookbook.Get(cookbook_name)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
					return
				}
				cb_ver, err := cb.GetVersion(cookbook_version)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
					return
				}
				if r.Method == "DELETE" {
					err := cb.DeleteVersion(cookbook_version)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
						return
					}
				} else {
					/* For cookbook version GET, we encode
					 * and return from here. It's... easier
					 * that way. */
					enc := json.NewEncoder(w)
					if err := enc.Encode(&cb_ver); err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					}
					return
				}
			case "PUT":
				/* First, see if the cookbook already exists, &
				 * if not create it. Second, see if this 
				 * specific version of the cookbook exists. If
				 * so, update it, otherwise, create it and set
				 * the latest version as needed. */
				cb, err := cookbook.Get(cookbook_name)
				if err != nil {
					cb, err = cookbook.New(cookbook_name)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				cbv, err := cb.GetVersion(cookbook_version)
				cbv_data, jerr := ParseObjJson(r.Body)
				if jerr != nil {
					JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				}
				if err != nil {
					_, err = cb.NewVersion(cookbook_version, cbv_data)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusCreated)
				} else {
					err := cbv.UpdateVersion(cbv_data)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
						return
					}
					cb.Save()
				}
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
