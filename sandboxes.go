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
	"encoding/json"
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
)

func sandboxHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathArray := splitPath(r.URL.Path)[2:]
	pathArrayLen := len(pathArray)
	sboxResponse := make(map[string]interface{})
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

	containerACL, conerr := acl.Get(org, "containers", "sandboxes")
	if conerr != nil {
		jsonErrorReport(w, r, conerr.Error(), conerr.Status())
		return
	}

	switch r.Method {
	case "POST":
		if pathArrayLen != 1 {
			jsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
			return
		}
		if f, err := containerACL.CheckPerm("create", opUser); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		}
		jsonReq, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		sboxHash, ok := jsonReq["checksums"].(map[string]interface{})
		if !ok {
			jsonErrorReport(w, r, "Field 'checksums' missing", http.StatusBadRequest)
			return
		} else if len(sboxHash) == 0 {
			jsonErrorReport(w, r, "Bad checksums!", http.StatusBadRequest)
			return
		} else {
			for _, j := range sboxHash {
				if j != nil {
					jsonErrorReport(w, r, "Bad checksums!", http.StatusBadRequest)
					return
				}
			}
		}
		sbox, err := sandbox.New(org, sboxHash)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		err = sbox.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}

		/* If we're here, make the slightly weird response. */
		sboxResponse["uri"] = util.ObjURL(sbox)
		sboxResponse["sandbox_id"] = sbox.ID
		sboxResponse["checksums"] = sbox.UploadChkList()
		w.WriteHeader(http.StatusCreated)
	case "PUT":
		if pathArrayLen != 2 {
			jsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
			return
		}
		if f, err := containerACL.CheckPerm("update", opUser); err != nil {
			jsonErrorReport(w, r, err.Error(), err.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		}

		sandboxID := vars["id"]

		jsonReq, jerr := parseObjJSON(r.Body)
		if jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}
		sboxCommit, ok := jsonReq["is_completed"].(bool)
		if !ok {
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
		}

		sbox, err := sandbox.Get(org, sandboxID)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		if err = sbox.IsComplete(); err == nil {
			sbox.Completed = sboxCommit
			err = sbox.Save()
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			jsonErrorReport(w, r, err.Error(), http.StatusServiceUnavailable)
			return
		}

		/* The response here is a bit confusing too. The
		 * documented behavior doesn't match with the observed
		 * behavior from chef-zero, and it's not real clear what
		 * it wants from the checksums array. Still, we need to
		 * give it what it wants. Ask about this later. */
		sboxResponse["guid"] = sbox.ID
		sboxResponse["name"] = sbox.ID
		sboxResponse["is_completed"] = sbox.Completed
		sboxResponse["create_time"] = sbox.CreationTime.UTC().Format("2006-01-02T15:04:05+00:00")
		sboxResponse["checksums"] = sbox.Checksums
	default:
		jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&sboxResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
