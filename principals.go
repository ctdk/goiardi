/* Principals functions */

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
	"github.com/ctdk/goiardi/actor"
	"net/http"
)

func principalHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principalName := r.URL.Path[12:]
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		// Can't so easily get out of the full request for a user or
		// client object here, so just go ahead and do that rather than
		// dancing around.
		chefActor, err := actor.GetReqUser(principalName)

		if r.Method == http.MethodHead {
			var status int
			if err != nil {
				status = http.StatusNotFound
			} else {
				status = http.StatusOK
			}
			headResponse(w, r, status)
			return
		}

		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}

		var chefType string
		if chefActor.IsUser() {
			chefType = "user"
		} else {
			chefType = "client"
		}
		jsonPrincipal := map[string]interface{}{
			"name":       chefActor.GetName(),
			"type":       chefType,
			"public_key": chefActor.PublicKey(),
		}
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&jsonPrincipal); encerr != nil {
			jsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Unrecognized method for principals!", http.StatusMethodNotAllowed)
	}
}
