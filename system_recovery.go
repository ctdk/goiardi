/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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
	"github.com/ctdk/goiardi/user"
	"log"
	"net/http"
)

func systemRecoveryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		jsonErrorReport(w, r, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	opUser, oerr := actor.GetReqUser(nil, r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if !opUser.IsAdmin() {
		jsonErrorReport(w, r, "missing create permission", http.StatusForbidden)
		return
	}

	userData, jerr := parseObjJSON(r.Body)
	if jerr != nil {
		jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
		return
	}
	auth, authErr := validateJSON(userData)
	if authErr != nil {
		jsonErrorReport(w, r, authErr.Error(), http.StatusBadRequest)
		return
	}
	resetter, rerr := user.Get(auth.Name)
	if rerr != nil {
		s := rerr.Status()
		msg := rerr.Error()
		if s == http.StatusNotFound {
			s = http.StatusForbidden
			msg = "System recovery disabled for this user"
		}
		jsonErrorReport(w, r, msg, s)
		return
	}
	if !resetter.Recoveror {
		jsonErrorReport(w, r, "System recovery disabled for this user", http.StatusForbidden)
		return
	}

	resp := make(map[string]interface{})
	resp["display_name"] = resetter.Name
	resp["email"] = resetter.Email
	resp["username"] = resetter.Username
	resp["recovery_authentication_enabled"] = resetter.Recoveror

	perr := resetter.CheckPasswd(auth.Password)
	if perr != nil {
		jsonErrorReport(w, r, "Failed to authenticate: Username and password incorrect", http.StatusUnauthorized)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
	log.Printf("json data for system_recovery: %+v", userData)
	return
}
