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
	"log"
	"net/http"
	"encoding/json"
	"bytes"
	"compress/gzip"
	"io"
)

func report_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("URL: %s", r.URL.Path)
	log.Printf("encoding %s", r.Header.Get("Content-Encoding"))

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
	var report_response interface{}

	switch r.Method {
		case "GET":
			// Making an informed guess that admin rights are needed
			// to see the node run reports
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
				runs := report.GetNodeList(nodeName)
				// try sending it back as just an array
				report_response = runs
			} else if op == "org" {
				if runId, ok := path_array[3]; ok {
					run, err := report.Get(runId)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), err.Status())
						return
					}
					report_response = run
				} else {
					runs := report.GetReportList()
					report_response = runs
				}
			} else {
				JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
				return
			}
		case "POST":
			json_report, jerr := ParseObjJson(r.Body)
			if jerr != nil {
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
				err = rep.Save()
				report_response = rep
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
				err = rep.Save()
				if err != nil {
					JsonErrorReport(w, r, err.Error(), err.Status())
					return
				}
				// .... and?
				report_response = rep
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
