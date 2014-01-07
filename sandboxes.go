/* Sandbox functions */

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
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/util"
)

func sandbox_handler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	path_array := SplitPath(r.URL.Path)
	sbox_response := make(map[string]interface{})

	switch r.Method {
		case "POST":
			if len(path_array) != 1 {
				JsonErrorReport(w, r, "Bad request.", http.StatusBadRequest)
				return
			}
			json_req, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			sbox_hash, ok := json_req["checksums"].(map[string]interface{})
			if !ok {
				JsonErrorReport(w, r, "Field 'checksums' missing", http.StatusBadRequest)
				return
			} else if len(sbox_hash) == 0 {
				JsonErrorReport(w, r, "Bad checksums!", http.StatusBadRequest)
				return
			} else {
				for _, j := range sbox_hash {
					if j != nil {
						JsonErrorReport(w, r, "Bad checksums!", http.StatusBadRequest)
						return
					}
				}
			}
			sbox, err := sandbox.New(sbox_hash)
			sbox.Save()
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return 
			}
			/* If we're here, make the slightly weird response. */
			sbox_response["uri"] = util.ObjURL(sbox)
			sbox_response["sandbox_id"] = sbox.Id
			sbox_response["checksums"] = sbox.UploadChkList()
			w.WriteHeader(http.StatusCreated)
		case "PUT":
			if len(path_array) != 2 {
				JsonErrorReport(w, r, "Bad request.", http.StatusBadRequest)
				return
			}

			sandbox_id := path_array[1]
			
			json_req, jerr := ParseObjJson(r.Body)
			if jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}
			sbox_commit, ok := json_req["is_completed"].(bool)
			if !ok {
				JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}

			sbox, err := sandbox.Get(sandbox_id)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
				return
			}

			if err = sbox.IsComplete(); err == nil {
				sbox.Completed = sbox_commit
				sbox.Save()
			} else {
				JsonErrorReport(w, r, err.Error(), http.StatusServiceUnavailable)
				return
			}

			/* The response here is a bit confusing too. The
			 * documented behavior doesn't match with the observed
			 * behavior from chef-zero, and it's not real clear what
			 * it wants from the checksums array. Still, we need to
			 * give it what it wants. Ask about this later. */
			sbox_response["guid"] = sbox.Id
			sbox_response["name"] = sbox.Id
			sbox_response["is_completed"] = sbox.Completed
			sbox_response["create_time"] = sbox.CreationTime.UTC().Format("2006-01-02T15:04:05+00:00")
			sbox_response["checksums"] = sbox.Checksums
		default:
			JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&sbox_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
