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
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/cookbook"
	"net/http"
)

func universeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "GET" {
		jsonErrorReport(w, r, "Unrecognized method", http.StatusMethodNotAllowed)
		return
	}
	var noCache, force string
	r.ParseForm()
	if f, fok := r.Form["no-cache"]; fok {
		if len(f) > 0 {
			noCache = f[0]
		}
	}
	if f, fok := r.Form["force"]; fok {
		if len(f) > 0 {
			force = f[0]
		}
	}

	var universe map[string]map[string]interface{}
	cacheHeader := "NO"
	if force == "true" {
		go cookbook.UpdateUniverseCache()
	} else if noCache != "true" {
		universe = cookbook.UniverseCached()
		if universe == nil {
			go cookbook.UpdateUniverseCache()
		} else {
			cacheHeader = "YES"
		}
	}
	w.Header().Set("X-UNIVERSE-CACHE", cacheHeader)
	// Either noCache was false or there was nothing in the cache.
	if universe == nil {
		logger.Debugf("universe was nil, noCache was %s", noCache)
		universe = cookbook.Universe()
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&universe); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}
