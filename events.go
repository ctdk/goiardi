/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Serve up the list of logged events, and individual events as well.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"strconv"
)

// The whole list
func eventListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	// Look for offset and limit parameters
	r.ParseForm()
	var offset, limit, purgeFrom int
	if o, found := r.Form["offset"]; found {
		if len(o) < 0 {
			jsonErrorReport(w, r, "invalid offsets", http.StatusBadRequest)
			return
		}
		var err error
		offset, err = strconv.Atoi(o[0])
		if err != nil {
			jsonErrorReport(w, r, "invalid offset conversion to int", http.StatusBadRequest)
			return
		}
		if offset < 0 {
			jsonErrorReport(w, r, "invalid negative offset value", http.StatusBadRequest)
			return
		}
	} else {
		offset = 0
	}
	var limitFound bool
	if l, found := r.Form["limit"]; found {
		limitFound = true
		if len(l) < 0 {
			jsonErrorReport(w, r, "invalid limit", http.StatusBadRequest)
			return
		}
		var err error
		limit, err = strconv.Atoi(l[0])
		if err != nil {
			jsonErrorReport(w, r, "invalid limit converstion to int", http.StatusBadRequest)
			return
		}
		if limit < 0 {
			jsonErrorReport(w, r, "invalid negative limit value", http.StatusBadRequest)
			return
		}
	}

	if p, found := r.Form["purge"]; found {
		if len(p) < 0 {
			jsonErrorReport(w, r, "invalid purge id", http.StatusBadRequest)
			return
		}
		var err error
		purgeFrom, err = strconv.Atoi(p[0])
		if err != nil {
			jsonErrorReport(w, r, "invalid purge from converstion to int", http.StatusBadRequest)
			return
		}
		if purgeFrom < 0 {
			jsonErrorReport(w, r, "invalid negative purgeFrom value", http.StatusBadRequest)
			return
		}
	}

	paramStrs := []string{"from", "until", "action", "object_type", "object_name", "doer"}
	searchParams := make(map[string]string, 6)

	for _, v := range paramStrs {
		if st, found := r.Form[v]; found {
			if len(st) < 0 {
				jsonErrorReport(w, r, "invalid "+v, http.StatusBadRequest)
				return
			}
			searchParams[v] = st[0]
		}
	}

	switch r.Method {
	case "GET":
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You must be an admin to do that", http.StatusForbidden)
			return
		}
		var leList []*loginfo.LogInfo
		var err error
		if limitFound {
			leList, err = loginfo.GetLogInfos(searchParams, offset, limit)
		} else {
			leList, err = loginfo.GetLogInfos(searchParams, offset)
		}
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		leResp := make([]map[string]interface{}, len(leList))
		for i, v := range leList {
			leResp[i] = make(map[string]interface{})
			leResp[i]["event"] = v
			leURL := fmt.Sprintf("/events/%d", v.ID)
			leResp[i]["url"] = util.CustomURL(leURL)
		}
		enc := json.NewEncoder(w)
		if err := enc.Encode(&leResp); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case "DELETE":
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You must be an admin to do that", http.StatusForbidden)
			return
		}
		purged, err := loginfo.PurgeLogInfos(purgeFrom)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
		}
		leResp := make(map[string]string)
		leResp["purged"] = fmt.Sprintf("Purged %d logged events", purged)
		enc := json.NewEncoder(w)
		if err := enc.Encode(&leResp); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// Individual log events
func eventHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	eventID, aerr := strconv.Atoi(r.URL.Path[8:])
	if aerr != nil {
		jsonErrorReport(w, r, aerr.Error(), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You must be an admin to do that", http.StatusForbidden)
			return
		}
		le, err := loginfo.Get(eventID)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(&le); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case "DELETE":
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You must be an admin to do that", http.StatusForbidden)
			return
		}
		le, err := loginfo.Get(eventID)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		err = le.Delete()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(&le); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	return
}
