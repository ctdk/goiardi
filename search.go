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
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/search"
	"github.com/ctdk/goiardi/data_bag"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/data_store"
	"net/http"
	"encoding/json"
	"fmt"
	"strconv"
	"regexp"
	"log"
)

func search_handler(w http.ResponseWriter, r *http.Request){
	/* ... and we need search to run the environment tests, so here we
	 * go. */
	w.Header().Set("Content-Type", "application/json")
	search_response := make(map[string]interface{})
	path_array := SplitPath(r.URL.Path)
	path_array_len := len(path_array)

	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}

	/* set up query params for searching */
	var (
		paramQuery string
		paramsRows int
		sortOrder string
		start int
	)
	r.ParseForm()
	if q, found := r.Form["q"]; found {
		if len(q) < 0 {
			JsonErrorReport(w, r, "No query string specified for search", http.StatusBadRequest)
			return
		}
		paramQuery = q[0]
	} else if path_array_len != 1 {
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
	// TODO: get a default sort in, once sorting is sorted out.
	if s, found := r.Form["sort"]; found {
		if len(s) > 0 {
			sortOrder = s[0]
		} else {
			sortOrder = "id ASC"
		}
	}
	_ = sortOrder
	if st, found := r.Form["start"]; found {
		if len(st) > 0 {
			start, _ = strconv.Atoi(st[0])
		}
	} else {
		start = 0
	}

	if path_array_len == 1 {
		/* base end points */
		switch r.Method {
			case "GET":
				if opUser.IsValidator() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				searchEndpoints := search.GetEndpoints()
				for _, s := range searchEndpoints {
					search_response[s] = util.CustomURL(fmt.Sprintf("/search/%s", s))
				}
			default:
				JsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
				return
		}
	} else if path_array_len == 2 {
		switch r.Method {
			case "GET", "POST":
				if opUser.IsValidator() {
					JsonErrorReport(w, r, "You are not allowed to perform this action", http.StatusForbidden)
					return
				}
				/* start figuring out what comes in POSTS now,
				 * so the partial search tests don't complain
				 * anymore. */
				var partial_data map[string]interface{}
				if r.Method == "POST" {
					var perr error
					partial_data, perr = ParseObjJson(r.Body)
					if perr != nil {
						JsonErrorReport(w, r, perr.Error(), http.StatusBadRequest)
						return
					}
				}

				idx := path_array[1]
				rObjs, err := search.Search(idx, paramQuery)

				if err != nil {
					statusCode := http.StatusBadRequest
					re := regexp.MustCompile(`^I don't know how to search for .*? data objects.`)
					if re.MatchString(err.Error()) {
						statusCode = http.StatusNotFound
					}
					JsonErrorReport(w, r, err.Error(), statusCode)
					return
				}

				res := make([]map[string]interface{}, len(rObjs))
				for i, r := range rObjs {
					switch r := r.(type) {
						case *client.Client:
							jc := map[string]interface{}{
								"name": r.Name,
								"chef_type": r.ChefType,
								"json_class": r.JsonClass,
								"admin": r.Admin,
								"public_key": r.PublicKey,
								"validator": r.Validator,
							}
							res[i] = jc
						default:
							res[i] = util.MapifyObject(r)
					}
				}

				/* If we're doing partial search, tease out the
				 * fields we want. */
				if r.Method == "POST" {
					res, err = partialSearchFormat(res, partial_data)
					if err != nil {
						JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
						return
					}
					for x, z := range res {
						tmpRes := make(map[string]interface{})
						switch ro := rObjs[x].(type) {
							case *data_bag.DataBagItem:
								dbi_url := fmt.Sprintf("/data/%s/%s", ro.DataBagName, ro.RawData["id"].(string))
								tmpRes["url"] = util.CustomURL(dbi_url)
							default:
								tmpRes["url"] = util.ObjURL(rObjs[x].(util.GoiardiObj))
							}
						tmpRes["data"] = z

						res[x] = tmpRes
					}
				}
				
				end := start + paramsRows
				if end > len(res) {
					end = len(res)
				}
				res = res[start:end]
				search_response["total"] = len(res)
				search_response["start"] = start
				search_response["rows"] = res
			default:
				JsonErrorReport(w, r, "Method not allowed", http.StatusMethodNotAllowed)
				return
		}
	} else {
		/* Say what? Bad request. */
		JsonErrorReport(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(&search_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
}

func reindexHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	reindex_response := make(map[string]interface{})
	opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
	if oerr != nil {
		JsonErrorReport(w, r, oerr.Error(), oerr.Status())
		return
	}
	switch r.Method {
		case "POST":
			if !opUser.IsAdmin() {
				JsonErrorReport(w, r, "You are not allowed to perform that action.", http.StatusForbidden)
				return
			}
			reindexObjs := make([]indexer.Indexable, 0)
			// We clear the index, *then* do the fetch because if
			// something comes in between the time we fetch the
			// objects to reindex and when it gets done, they'll
			// just be added naturally
			indexer.ClearIndex()
			// default indices
			defaults := [...]string{ "node", "client", "role", "env" }
			ds := data_store.New()
			for _, d := range defaults {
				objList := ds.GetList(d)
				for _, oname := range objList {
					u, _ := ds.Get(d, oname)
					if u != nil {
						reindexObjs = append(reindexObjs, u.(indexer.Indexable))	
					}
				}
			}
			// data bags have to be done separately
			dbags := data_bag.GetList()
			for _, db := range dbags {
				dbag, err := data_bag.Get(db)
				if err != nil {
					continue
				}
				dbis := make([]indexer.Indexable, dbag.NumDBItems())
				i := 0
				allDBItems, derr := dbag.AllDBItems()
				if derr != nil {
					log.Println(derr)
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
			reindex_response["reindex"] = "OK"
		default:
			JsonErrorReport(w, r, "Method not allowed. If you're trying to do something with a data bag named 'reindex', it's not going to work I'm afraid.", http.StatusMethodNotAllowed)
			return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&reindex_response); err != nil {
		JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
	}
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
						keyRange := []string{ "raw_data", "default", "default_attributes", "normal", "override", "override_attributes", "automatic" }
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

func walk (v interface{}, keys []string) (interface{}) {
	switch v := v.(type) {
		case map[string]interface{}:
			if _, found := v[keys[0]]; found {
				if len(keys) > 1 {
					return walk(v[keys[0]], keys[1:])
				} else {
					return v[keys[0]]
				}
			} else {
				return nil
			}
		case map[string]string:
			return v[keys[0]]
		case map[string][]string:
			return v[keys[0]]
		default:
			if len(keys) == 1 {
				return v
			} else {
				return nil
			}
	}
}
