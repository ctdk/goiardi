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
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/search"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type results struct {
	res []map[string]interface{}
	sortKey string
}

func (r results) Len() int { return len(r.res) }
func (r results) Swap(i, j int) { r.res[i], r.res[j] = r.res[j], r.res[i] }
func (r results) Less(i, j int) bool {
	ibase := r.res[i][r.sortKey]
	jbase := r.res[j][r.sortKey]
	ival := reflect.ValueOf(ibase)
	jval := reflect.ValueOf(jbase)
	if (!ival.IsValid() && !jval.IsValid()) || ival.IsValid() && !jval.IsValid() {
		return true
	} else if !ival.IsValid() && jval.IsValid() {
		return false
	}
	// don't try and compare different types for now. If this ever becomes
	// an issue in practice, though, it should be revisited
	if ival.Type() == jval.Type() {
		switch ibase.(type) {
			case int, int8, int32, int64:
				return ival.Int() < jval.Int()
			case uint, uint8, uint32, uint64:
				return ival.Uint() < jval.Uint()
			case float32, float64:
				return ival.Float() < jval.Float()
			case string:
				return ival.String() < jval.String()
		}
	}
	
	return false
}

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

	if pathArrayLen == 1 {
		/* base end points */
		switch r.Method {
		case "GET":
			if opUser.IsValidator() {
				jsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
				return
			}
			searchEndpoints := search.GetEndpoints()
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
			rObjs, err := search.Search(idx, paramQuery)

			if err != nil {
				statusCode := http.StatusBadRequest
				re := regexp.MustCompile(`^I don't know how to search for .*? data objects.`)
				if re.MatchString(err.Error()) {
					statusCode = http.StatusNotFound
				}
				jsonErrorReport(w, r, err.Error(), statusCode)
				return
			}

			res := make([]map[string]interface{}, len(rObjs))
			for i, r := range rObjs {
				switch r := r.(type) {
				case *client.Client:
					jc := map[string]interface{}{
						"name":       r.Name,
						"chef_type":  r.ChefType,
						"json_class": r.JSONClass,
						"admin":      r.Admin,
						"public_key": r.PublicKey(),
						"validator":  r.Validator,
					}
					res[i] = jc
				default:
					res[i] = util.MapifyObject(r)
				}
			}

			/* If we're doing partial search, tease out the
			 * fields we want. */
			if r.Method == "POST" {
				res, err = partialSearchFormat(res, partialData)
				if err != nil {
					jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
					return
				}
				for x, z := range res {
					tmpRes := make(map[string]interface{})
					switch ro := rObjs[x].(type) {
					case *databag.DataBagItem:
						dbiURL := fmt.Sprintf("/data/%s/%s", ro.DataBagName, ro.RawData["id"].(string))
						tmpRes["url"] = util.CustomURL(dbiURL)
					default:
						tmpRes["url"] = util.ObjURL(rObjs[x].(util.GoiardiObj))
					}
					tmpRes["data"] = z

					res[x] = tmpRes
				}
			}

			// and at long last, sort
			ss := strings.Split(sortOrder, " ")
			sortKey := ss[0]
			if sortKey == "id" {
				sortKey = "name"
			}
			var ordering string
			if len(ss) > 1 {
				ordering = strings.ToLower(ss[1])
			} else {
				ordering = "asc"
			}
			sortResults := results{ res, sortKey }
			if ordering == "desc" {
				sort.Sort(sort.Reverse(sortResults))
			} else {
				sort.Sort(sortResults)
			}
			res = sortResults.res

			end := start + paramsRows
			if end > len(res) {
				end = len(res)
			}
			res = res[start:end]
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

func partialSearchFormat(results []map[string]interface{}, partialFormat map[string]interface{}) ([]map[string]interface{}, error) {
	/* regularize partial search keys */
	psearchKeys := make(map[string][]string, len(partialFormat))
	for k, v := range partialFormat {
		switch v := v.(type) {
		case []interface{}:
			psearchKeys[k] = make([]string, len(v))
			for i, j := range v {
				switch j := j.(type) {
				case string:
					psearchKeys[k][i] = j
				default:
					err := fmt.Errorf("Partial search key %s badly formatted: %T %v", k, j, j)
					return nil, err
				}
			}
		case []string:
			psearchKeys[k] = make([]string, len(v))
			for i, j := range v {
				psearchKeys[k][i] = j
			}
		default:
			err := fmt.Errorf("Partial search key %s badly formatted: %T %v", k, v, v)
			return nil, err
		}
	}
	newResults := make([]map[string]interface{}, len(results))

	for i, j := range results {
		newResults[i] = make(map[string]interface{})
		for key, vals := range psearchKeys {
			var pval interface{}
			/* The first key can either be top or first level.
			 * Annoying, but that's how it is. */
			if len(vals) > 0 {
				if step, found := j[vals[0]]; found {
					if len(vals) > 1 {
						pval = walk(step, vals[1:])
					} else {
						pval = step
					}
				} else {
					if len(vals) > 0 {
						// bear in mind precedence. We need to
						// overwrite later values with earlier
						// ones.
						keyRange := []string{"raw_data", "default", "default_attributes", "normal", "override", "override_attributes", "automatic"}
						for _, r := range keyRange {
							tval := walk(j[r], vals[0:])
							if tval != nil {
								switch pv := pval.(type) {
								case map[string]interface{}:
									// only merge if tval is also a map[string]interface{}
									switch tval := tval.(type) {
									case map[string]interface{}:
										for g, h := range tval {
											pv[g] = h
										}
										pval = pv
									}
								default:
									pval = tval
								}
							}
						}
					}
				}
			}
			newResults[i][key] = pval
		}
	}
	return newResults, nil
}

func walk(v interface{}, keys []string) interface{} {
	switch v := v.(type) {
	case map[string]interface{}:
		if _, found := v[keys[0]]; found {
			if len(keys) > 1 {
				return walk(v[keys[0]], keys[1:])
			}
			return v[keys[0]]
		}
		return nil
	case map[string]string:
		return v[keys[0]]
	case map[string][]string:
		return v[keys[0]]
	default:
		if len(keys) == 1 {
			return v
		}
		return nil
	}
}
