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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/report"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func reportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	protocolVersion := r.Header.Get("X-Ops-Reporting-Protocol-Version")
	if protocolVersion == "" {
		// try a param (makes working with webui easier)
		form, e := url.ParseQuery(r.URL.RawQuery)
		if e != nil {
			jsonErrorReport(w, r, e.Error(), http.StatusBadRequest)
			return
		}
		if p, f := form["protocol-version"]; f {
			if len(p) > 0 {
				protocolVersion = p[0]
			}
		}
	}
	// someday there may be other protocol versions
	if protocolVersion != "0.1.0" {
		jsonErrorReport(w, r, "Unsupported reporting protocol version", http.StatusNotFound)
		return
	}

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	// TODO: some params for time ranges exist and need to be handled
	// properly

	pathArray := splitPath(r.URL.Path)
	pathArrayLen := len(pathArray)
	reportResponse := make(map[string]interface{})

	switch r.Method {
	case "GET":
		// Making an informed guess that admin rights are needed
		// to see the node run reports
		r.ParseForm()
		var rows int
		var from, until time.Time
		var status string
		if fr, found := r.Form["rows"]; found {
			if len(fr) < 0 {
				jsonErrorReport(w, r, "invalid rows", http.StatusBadRequest)
				return
			}
			var err error
			rows, err = strconv.Atoi(fr[0])
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			// default is 10
			rows = 10
		}
		if ff, found := r.Form["from"]; found {
			if len(ff) < 0 {
				jsonErrorReport(w, r, "invalid from", http.StatusBadRequest)
				return
			}
			fromUnix, err := strconv.ParseInt(ff[0], 10, 64)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return
			}
			from = time.Unix(fromUnix, 0)
		} else {
			from = time.Now().Add(-(time.Duration(24*90) * time.Hour))
		}
		if fu, found := r.Form["until"]; found {
			if len(fu) < 0 {
				jsonErrorReport(w, r, "invalid until", http.StatusBadRequest)
				return
			}
			untilUnix, err := strconv.ParseInt(fu[0], 10, 64)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
				return
			}
			until = time.Unix(untilUnix, 0)
		} else {
			until = time.Now()
		}

		if st, found := r.Form["status"]; found {
			if len(st) < 0 {
				jsonErrorReport(w, r, "invalid status", http.StatusBadRequest)
				return
			}
			status = st[0]
			if status != "started" && status != "success" && status != "failure" {
				jsonErrorReport(w, r, "invalid status given", http.StatusBadRequest)
				return
			}
		}

		// If the end time is more than 90 days ahead of the
		// start time, give an error
		if from.Truncate(time.Hour).Sub(until.Truncate(time.Hour)) >= (time.Duration(24*90) * time.Hour) {
			msg := fmt.Sprintf("End time %s is too far ahead of start time %s (max 90 days)", until.String(), from.String())
			jsonErrorReport(w, r, msg, http.StatusNotAcceptable)
			return
		}

		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
			return
		}
		if pathArrayLen < 3 || pathArrayLen > 4 {
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
		}
		op := pathArray[1]
		if op == "nodes" && pathArrayLen == 4 {
			nodeName := pathArray[2]
			runs, nerr := report.GetNodeList(nodeName, from, until, rows, status)
			if nerr != nil {
				jsonErrorReport(w, r, nerr.Error(), http.StatusInternalServerError)
				return
			}
			reportResponse["run_history"] = runs
		} else if op == "org" {
			if pathArrayLen == 4 {
				runID := pathArray[3]
				run, err := report.Get(runID)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				reportResponse = formatRunShow(run)
			} else {
				runs, rerr := report.GetReportList(from, until, rows, status)
				if rerr != nil {
					jsonErrorReport(w, r, rerr.Error(), http.StatusInternalServerError)
					return
				}
				reportResponse["run_history"] = runs
			}
		} else {
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
		}
	case "POST":
		// Can't use the usual parseObjJSON function here, since
		// the reporting "run_list" type is a string rather
		// than []interface{}.
		jsonReport := make(map[string]interface{})
		dec := json.NewDecoder(r.Body)
		dec.UseNumber()
		if jerr := dec.Decode(&jsonReport); jerr != nil {
			jsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
			return
		}

		if pathArrayLen < 4 || pathArrayLen > 5 {
			jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
		}
		nodeName := pathArray[2]
		if pathArrayLen == 4 {
			rep, err := report.NewFromJSON(nodeName, jsonReport)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			// what's the expected response?
			serr := rep.Save()
			if serr != nil {
				jsonErrorReport(w, r, serr.Error(), http.StatusInternalServerError)
				return
			}
			reportResponse["run_detail"] = rep
		} else {
			runID := pathArray[4]
			rep, err := report.Get(runID)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			err = rep.UpdateFromJSON(jsonReport)
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
			serr := rep.Save()
			if serr != nil {
				jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			// .... and?
			reportResponse["run_detail"] = rep
		}
	default:
		jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&reportResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

// This function is subject to change, depending on what the client actually
// expects. This may not be entirely correct.
func formatRunShow(run *report.Report) map[string]interface{} {
	reportMap := util.MapifyObject(run)
	resources := reportMap["resources"]
	delete(reportMap, "resources")
	reportFmt := make(map[string]interface{})
	reportFmt["run_detail"] = reportMap
	reportFmt["resources"] = resources
	return reportFmt
}
