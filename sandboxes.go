/* Sandbox functions */

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
	"net/http"

	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/util"
)

func sandboxHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathArray := splitPath(r.URL.Path)
	sboxResponse := make(map[string]interface{})
	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	switch r.Method {
	case http.MethodPost:
		if len(pathArray) != 1 {
			jsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
			return
		}
		if !opUser.IsAdmin() {
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
		sbox, err := sandbox.New(sboxHash)
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
	case http.MethodPut:
		if len(pathArray) != 2 {
			jsonErrorReport(w, r, "Bad request.", http.StatusMethodNotAllowed)
			return
		}
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You are not allowed to take this action.", http.StatusForbidden)
			return
		}

		sandboxID := pathArray[1]

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

		sbox, err := sandbox.Get(sandboxID)
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
