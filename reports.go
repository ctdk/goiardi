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
	if r.Method == "POST" {
		//json_req, err := ParseObjJson(r.Body)
		var reader io.ReadCloser
		switch r.Header.Get("Content-Encoding") {
			case "gzip":
				var err error
				reader, err = gzip.NewReader(r.Body)
				if err != nil {
					JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
			default:
				reader = r.Body
		} 
		var buf bytes.Buffer
		_, err := buf.ReadFrom(reader)
		if err != nil {
			JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("body of request")
		//log.Printf("%+v", json_req)
		s := buf.Bytes()
		log.Printf("%s", string(s))
		log.Printf("%v", s)
	}
	report_response := make(map[string]string)
	report_response["msg"] = "ok then"
	enc := json.NewEncoder(w)
	if err := enc.Encode(&report_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
