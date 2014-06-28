/* Principals functions */

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
	"github.com/ctdk/goiardi/actor"
	"net/http"
)

func principal_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	principal_name := r.URL.Path[12:]
	switch r.Method {
	case "GET":
		chef_actor, err := actor.GetReqUser(principal_name)
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		var chefType string
		if chef_actor.IsUser() {
			chefType = "user"
		} else {
			chefType = "client"
		}
		json_principal := map[string]interface{}{
			"name":       chef_actor.GetName(),
			"type":       chefType,
			"public_key": chef_actor.PublicKey(),
		}
		enc := json.NewEncoder(w)
		if encerr := enc.Encode(&json_principal); encerr != nil {
			JsonErrorReport(w, r, encerr.Error(), http.StatusInternalServerError)
			return
		}
	default:
		JsonErrorReport(w, r, "Unrecognized method for principals!", http.StatusMethodNotAllowed)
	}
}
