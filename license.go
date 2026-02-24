/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"net/http"
)

// Handle requests for "licensing" info. Heh.

func licenseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		jsonErrorReport(w, r, "Unrecognized method for client", http.StatusMethodNotAllowed)
		return
	}
	resp := map[string]interface{}{
		"limit_exceeded": false,
		"node_license": 1000000000,
		"node_count": 0,
		"upgrade_url": "https://github.com/ctdk/goiardi/blob/master/LICENSE",
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&resp); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		return
	}
}
