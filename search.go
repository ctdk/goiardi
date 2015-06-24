/* Search functions */

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
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/search"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

func searchHandler(w http.ResponseWriter, r *http.Request) {
	/* ... and we need search to run the environment tests, so here we
	 * go. */
	w.Header().Set("Content-Type", "application/json")
	searchResponse := make(map[string]interface{})
	pathArray := splitPath(r.URL.Path)
	pathArrayLen := len(pathArray)

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	/* set up query params for searching */
	var (
		paramQuery string
		paramsRows int
		sortOrder  string
		start      int
	)
	r.ParseForm()
	if q, found := r.Form["q"]; found {
		if len(q) < 0 {
			jsonErrorReport(w, r, "No query string specified for search", http.StatusBadRequest)
			return
		}
		paramQuery = q[0]
	} else if pathArrayLen != 1 {
		/* default to "*:*" for a search term */
		paramQuery = "*:*"
	}
	if pr, found := r.Form["rows"]; found {
		if len(pr) > 0 {
			paramsRows, _ = strconv.Atoi(pr[0])
		}
	} else {
		paramsRows = 1000
	}
	sortOrder = "id ASC"
	if s, found := r.Form["sort"]; found {
		if len(s) > 0 {
			if s[0] != "" {
				sortOrder = s[0]
			}
		} else {
			sortOrder = "id ASC"
		}
	}
	if st, found := r.Form["start"]; found {
		if len(st) > 0 {
			start, _ = strconv.Atoi(st[0])
		}
	} else {
		start = 0
	}

	var searcher search.Searcher
	if config.Config.PgSearch {
		searcher = &search.PostgresSearch{}
	} else {
		searcher = &search.TrieSearch{}
	}

	if pathArrayLen == 1 {
		/* base end points */
		switch r.Method {
		case "GET":
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			searchEndpoints := searcher.GetEndpoints()
			for _, s := range searchEndpoints {
				searchResponse[s] = util.CustomURL(fmt.Sprintf("/search/%s", s))
			}
		default:
			jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	} else if pathArrayLen == 2 {
		switch r.Method {
		case "GET", "POST":
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			var qerr error
			paramQuery, qerr = url.QueryUnescape(paramQuery)
			if qerr != nil {
				jsonErrorReport(w, r, qerr.Error(), http.StatusBadRequest)
				return
			}
			/* start figuring out what comes in POSTS now,
			 * so the partial search tests don't complain
			 * anymore. */
			var partialData map[string]interface{}
			if r.Method == "POST" {
				var perr error
				partialData, perr = parseObjJSON(r.Body)
				if perr != nil {
					jsonErrorReport(w, r, perr.Error(), http.StatusBadRequest)
					return
				}
			}

			idx := pathArray[1]
			res, err := searcher.Search(idx, paramQuery, paramsRows, sortOrder, start, partialData)

			if err != nil {
				statusCode := http.StatusBadRequest
				re := regexp.MustCompile(`^I don't know how to search for .*? data objects.`)
				if re.MatchString(err.Error()) {
					statusCode = http.StatusNotFound
				}
				jsonErrorReport(w, r, err.Error(), statusCode)
				return
			}

			searchResponse["total"] = len(res)
			searchResponse["start"] = start
			searchResponse["rows"] = res
		default:
			jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	} else {
		/* Say what? Bad request. */
		jsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&searchResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func reindexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reindexResponse := make(map[string]interface{})
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		jsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	switch r.Method {
	case "POST":
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}
		reindexAll()
		reindexResponse["reindex"] = "OK"
	default:
		jsonErrorReport(w, r, "Method not allowed. If you're trying to do something with a data bag named 'reindex', it's not going to work I'm afraid.", http.StatusMethodNotAllowed)
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&reindexResponse); err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func reindexAll() {
	reindexObjs := make([]indexer.Indexable, 0, 100)
	// We clear the index, *then* do the fetch because if
	// something comes in between the time we fetch the
	// objects to reindex and when it gets done, they'll
	// just be added naturally
	indexer.ClearIndex()

	for _, v := range client.AllClients() {
		reindexObjs = append(reindexObjs, v)
	}
	for _, v := range node.AllNodes() {
		reindexObjs = append(reindexObjs, v)
	}
	for _, v := range role.AllRoles() {
		reindexObjs = append(reindexObjs, v)
	}
	for _, v := range environment.AllEnvironments() {
		reindexObjs = append(reindexObjs, v)
	}
	defaultEnv, _ := environment.Get("_default")
	reindexObjs = append(reindexObjs, defaultEnv)
	// data bags have to be done separately
	dbags := databag.GetList()
	for _, db := range dbags {
		dbag, err := databag.Get(db)
		if err != nil {
			continue
		}
		dbis := make([]indexer.Indexable, dbag.NumDBItems())
		i := 0
		allDBItems, derr := dbag.AllDBItems()
		if derr != nil {
			logger.Errorf(derr.Error())
			continue
		}
		for _, k := range allDBItems {
			n := k
			dbis[i] = n
			i++
		}
		reindexObjs = append(reindexObjs, dbis...)
	}
	indexer.ReIndex(reindexObjs)
	return
}
