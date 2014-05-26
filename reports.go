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
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/report"
	"github.com/ctdk/goiardi/util"
	"strconv"
	"time"
	"fmt"
)

func report_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	protocol_version := r.Header.Get("X-Ops-Reporting-Protocol-Version")
	// someday there may be other protocol versions
	if protocol_version != "0.1.0" {
		JsonErrorReport(w, r, "Unsupported reporting protocol version", http.StatusNotFound)
		return
	}

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	// TODO: some params for time ranges exist and need to be handled
	// properly

	path_array := SplitPath(r.URL.Path)
	path_array_len := len(path_array)
	report_response := make(map[string]interface{})

	switch r.Method {
		case "GET":
			// Making an informed guess that admin rights are needed
			// to see the node run reports
			var rows int
			var from, until time.Time
			r.ParseForm()
			if fr, found := r.Form["rows"]; found {
				if len(fr) < 0 {
					JsonErrorReport(w, r, "invalid rows", http.StatusBadRequest)
					return
				}
				var err error
				rows, err = strconv.Atoi(fr[0])
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
			} else {
				// default is 10
				rows = 10 
			}
			if ff, found := r.Form["from"]; found {
				if len(ff) < 0 {
					JsonErrorReport(w, r, "invalid from", http.StatusBadRequest)
					return
				}
				fromUnix, err := strconv.ParseInt(ff[0], 10, 64)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
				from = time.Unix(fromUnix, 0)
			} else {
				from = time.Now().Add(-(time.Duration(24 * 90) * time.Hour))
			}
			if fu, found := r.Form["until"]; found {
				if len(fu) < 0 {
					JsonErrorReport(w, r, "invalid until", http.StatusBadRequest)
					return
				}
				untilUnix, err := strconv.ParseInt(fu[0], 10, 64)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
				until = time.Unix(untilUnix, 0)
			} else {
				until = time.Now()
			}

			// If the end time is more than 90 days ahead of the
			// start time, give an error
			if from.Truncate(time.Hour).Sub(until.Truncate(time.Hour)) >= (time.Duration(24 * 90) * time.Hour) {
				msg := fmt.Sprintf("End time %s is too far ahead of start time %s (max 90 days)", until.String(), from.String())
				JsonErrorReport(w, r, msg, http.StatusNotAcceptable)
				return
			}

			if !opUser.IsAdmin() {
				JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			if path_array_len < 3 || path_array_len > 4 {
				JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
			op := path_array[1]
			if op == "nodes" && path_array_len == 4 {
				nodeName := path_array[2]
				runs, nerr := report.GetNodeList(nodeName, from, until, rows)
				if nerr != nil {
					JsonErrorReport(w, r, nerr.Error(), http.StatusInternalServerError)
					return
				}
				report_response["run_history"] = runs
			} else if op == "org" {
				if path_array_len == 4 {
					runId := path_array[3]
					run, err := report.Get(runId)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					report_response = format_run_show(run)
				} else {
					runs, rerr := report.GetReportList(from, until, rows)
					if rerr != nil {
						JsonErrorReport(w, r, rerr.Error(), http.StatusInternalServerError)
						return
					}
					report_response["run_history"] = runs
				}
			} else {
				JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		case "POST":
			// Can't use the usual ParseObjJson function here, since
			// the reporting "run_list" type is a string rather
			// than []interface{}.
			json_report := make(map[string]interface{})
			dec := json.NewDecoder(r.Body)
			if jerr := dec.Decode(&json_report); jerr != nil {
				JsonErrorReport(w, r, jerr.Error(), http.StatusBadRequest)
				return
			}

			if path_array_len < 4 || path_array_len > 5 {
				JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
			nodeName := path_array[2]
			if path_array_len == 4 {
				rep, err := report.NewFromJson(nodeName, json_report)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				// what's the expected response?
				serr := rep.Save()
				if serr != nil {
					JsonErrorReport(w, r, serr.Error(), http.StatusInternalServerError)
					return
				}
				report_response["run_detail"] = rep
			} else {
				run_id := path_array[4]
				rep, err := report.Get(run_id)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				err = rep.UpdateFromJson(json_report)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				serr := rep.Save()
				if serr != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
					return
				}
				// .... and?
				report_response["run_detail"] = rep
			} 
		default:
			JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
			return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&report_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

// This function is subject to change, depending on what the client actually
// expects. This may not be entirely correct.
func format_run_show(run *report.Report) map[string]interface{} {
	report_map := util.MapifyObject(run)
	resources := report_map["resources"]
	delete(report_map, "resources")
	report_fmt := make(map[string]interface{})
	report_fmt["run_detail"] = report_map
	report_fmt["resources"] = resources
	return report_fmt
}
