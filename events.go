/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

// The whole list
func eventListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "log-infos", "read"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)

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
			jsonErrorReport(w, r, "invalid limit conversion to int", http.StatusBadRequest)
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
			jsonErrorReport(w, r, "invalid purge from conversion to int", http.StatusBadRequest)
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
	case http.MethodHead:
		headDefaultResponse(w, r)
		return
	case http.MethodGet:
		var leList []*loginfo.LogInfo
		var err error
		if limitFound {
			leList, err = loginfo.GetLogInfos(org, searchParams, offset, limit)
		} else {
			leList, err = loginfo.GetLogInfos(org, searchParams, offset)
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
	case http.MethodDelete:
		if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "log-infos", "delete"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		purged, err := loginfo.PurgeLogInfos(org, purgeFrom)
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
	vars := mux.Vars(r)
	org, orgerr := orgloader.Get(vars["org"])
	if orgerr != nil {
		jsonErrorReport(w, r, orgerr.Error(), orgerr.Status())
		return
	}
	opUser, oerr := reqctx.CtxReqUser(r.Context())
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "log-infos", "read"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)

		return
	}
	eventIDstr := vars["id"]
	eventID, aerr := strconv.Atoi(vars["id"])
	if aerr != nil {
		jsonErrorReport(w, r, aerr.Error(), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodHead:
		pcheck := func(r *http.Request, eventIDstr string, opUser actor.Actor) util.Gerror {
			// opUser.IsAdmin() call removed here. It was checking
			// to see if the opUser was not an admin, and returning
			// 403 if they weren't and were trying to do a HEAD
			// request. If the user doesn't have at least read
			// access to the events, though, they wouldn't even be
			// able to get here.
			return nil
		}
		headChecking(w, r, opUser, org, eventIDstr, loginfo.DoesExist, pcheck)
		return
	case http.MethodGet:
		le, err := loginfo.Get(org, eventID)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusNotFound)
			return
		}
		enc := json.NewEncoder(w)
		if err = enc.Encode(&le); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
	case http.MethodDelete:
		if f, ferr := org.PermCheck.CheckContainerPerm(opUser, "log-infos", "delete"); ferr != nil {
			jsonErrorReport(w, r, ferr.Error(), ferr.Status())
			return
		} else if !f {
			jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
			return
		}
		le, err := loginfo.Get(org, eventID)
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
