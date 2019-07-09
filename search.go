/* Search functions */

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

package main

import (
	"encoding/json"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/search"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	"github.com/tideland/golib/logger"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"sync"
)

const ReindexableTypes = 5

var riM *sync.Mutex
var reindexNum = 0
var pid int

func init() {
	pid = os.Getpid()
	riM = new(sync.Mutex)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	/* ... and we need search to run the environment tests, so here we
	 * go. */
	w.Header().Set("Content-Type", "application/json")

	searchResponse := make(map[string]interface{})
	pathArray := splitPath(r.URL.Path)[2:]
	pathArrayLen := len(pathArray)

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

	// if it's a HEAD response, just send back 200 no matter what, there's
	// no meaningful way to use HEAD with search that I can see
	if r.Method == http.MethodHead {
		if opUser.IsValidator() {
			headResponse(w, r, http.StatusForbidden)
			return
		}
		headDefaultResponse(w, r)
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
		case http.MethodGet:
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			searchEndpoints := searcher.GetEndpoints(org)
			for _, s := range searchEndpoints {
				searchResponse[s] = util.CustomURL(util.JoinStr("/organizations/", org.Name, "/search/", s))
			}
		default:
			jsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	} else if pathArrayLen == 2 {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
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
			if r.Method == http.MethodPost {
				var perr error
				partialData, perr = parseObjJSON(r.Body)
				if perr != nil {
					jsonErrorReport(w, r, perr.Error(), http.StatusBadRequest)
					return
				}
			}

			idx := pathArray[1]
			res, err := searcher.Search(org, idx, paramQuery, paramsRows, sortOrder, start, partialData)

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
	if f, ferr := org.PermCheck.RootCheckPerm(opUser, "update"); ferr != nil {
		jsonErrorReport(w, r, ferr.Error(), ferr.Status())
		return
	} else if !f {
		jsonErrorReport(w, r, "You do not have permission to do that", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodPost:
		if !opUser.IsAdmin() {
			jsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
			return
		}
		go reindexAll()
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

// TODO: This needs to be able to be done per-org.
func reindexAll() {
	// Take the mutex before starting to reindex everything. This way at
	// least reindexing jobs won't pile up on top of each other all trying
	// to execute simultaneously.
	rdex := reindexNum
	reindexNum++
	logger.Infof("Taking mutex for reindex %d ($$ %d)", rdex, pid)
	riM.Lock()
	logger.Infof("mutex acquired %d ($$ %d)", rdex, pid)
	rCh := make(chan struct{}, ReindexableTypes)
	defer func() {
		for u := 0; u < ReindexableTypes; u++ {
			<-rCh
			logger.Debugf("a reindexing goroutine finished")
		}
		logger.Infof("all reindexing goroutines finished, release reindexing mutex for %d ($$ %d)", rdex, pid)
		riM.Unlock()
		logger.Debugf("reindexing mutex for %d ($$ %d) unlocked", rdex, pid)
	}()

	// We clear the index, *then* do the fetch because if
	// something comes in between the time we fetch the
	// objects to reindex and when it gets done, they'll
	// just be added naturally
	logger.Infof("Beginning org search schema reindexing now")
	orgs, _ := orgloader.AllOrganizations()
	for _, org := range orgs {
		logger.Debugf("Clearing %s search schema and tables", org.Name)
		indexer.ClearIndex(org.Name)
		logger.Debugf("Starting to reindex org %s", org.Name)
		indexer.CreateOrgDex(org.Name)
		// Send the objects to be reindexed in somewhat more manageable chunks
		clientObjs := make([]indexer.Indexable, 0, 100)
		for _, v := range client.AllClients(org) {
			clientObjs = append(clientObjs, v)
		}
		logger.Debugf("reindexing %s clients", org.Name)
		indexer.ReIndex(clientObjs, rCh)

		nodeObjs := make([]indexer.Indexable, 0, 100)
		for _, v := range node.AllNodes(org) {
			nodeObjs = append(nodeObjs, v)
		}
		logger.Debugf("reindexing %s nodes", org.Name)
		indexer.ReIndex(nodeObjs, rCh)

		roleObjs := make([]indexer.Indexable, 0, 100)
		for _, v := range role.AllRoles(org) {
			roleObjs = append(roleObjs, v)
		}

		logger.Debugf("reindexing %s roles", org.Name)
		indexer.ReIndex(roleObjs, rCh)

		environmentObjs := make([]indexer.Indexable, 0, 100)
		for _, v := range environment.AllEnvironments(org) {
			environmentObjs = append(environmentObjs, v)
		}
		defaultEnv, _ := environment.Get(org, "_default")
		environmentObjs = append(environmentObjs, defaultEnv)
		logger.Debugf("reindexing environments %s", org.Name)
		indexer.ReIndex(environmentObjs, rCh)

		dbagObjs := make([]indexer.Indexable, 0, 100)
		// data bags have to be done separately
		dbags := databag.GetList(org)
		for _, db := range dbags {
			dbag, err := databag.Get(org, db)
			if err != nil {
				continue
			}
			// Don't forget to create the collections, because we
			// weren't for the postgres search index. (Somehow a
			// regression snuck in here, but what do you do?
			indexer.CreateNewCollection(org.Name, dbag.GetName())
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
			dbagObjs = append(dbagObjs, dbis...)
		}
		logger.Debugf("Reindexing %s data bags", org.Name)
		indexer.ReIndex(dbagObjs, rCh)
	}
	return
}
