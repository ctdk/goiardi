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
	"github.com/ctdk/goiardi/organization"
	"net/http"
)

func orgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	pathArray := splitPath(r.URL.Path)
	pathArrayLen := len(pathArray)

	// If pathArrayLen is greater than 2, this gets handed off to another
	// handler.
	if pathArrayLen > 2 {

	}

	// Otherwise, it's org work.
	op := pathArray[2]
	orgName := pathArray[1]

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	org, err := organization.Get(orgName)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
		return
	}
	// check for basic rights to the organization in question, before any
	// beefier checks further down.
	err = org.CheckActor(opUser)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), err.Status())
	}

	switch op {

	}
}
