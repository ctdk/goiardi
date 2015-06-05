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

// Package search provides search and index capabilities for goiardi.
package search

import (
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/util"
	"reflect"
	"sort"
	"strings"
)

// Searcher is an interface that any search backend needs to implement. It's
// up to the Searcher to use whatever backend it wants to return the desired
// results.
type Searcher interface {
	Search(string, string, int, string, int, map[string]interface{}) ([]map[string]interface{}, error)
	GetEndpoints() []string
}

type results struct {
	res     []map[string]interface{}
	sortKey string
}

func (r results) Len() int      { return len(r.res) }
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

// SolrQuery holds a parsed query and query chain to run against the index. It's
// called SolrQuery because the search queries use a subset of Solr's syntax.
type SolrQuery struct {
	queryChain Queryable
	idxName    string
	docs       map[string]indexer.Document
}

type TrieSearch struct {
}

// Search parses the given query string and search the given index for any
// matching results.
func (t *TrieSearch) Search(idx string, query string, rows int, sortOrder string, start int, partialData map[string]interface{}) ([]map[string]interface{}, error) {
	qq := &Tokenizer{Buffer: query}
	qq.Init()
	if err := qq.Parse(); err != nil {
		return nil, err
	}
	qq.Execute()
	qchain := qq.Evaluate()
	d := make(map[string]indexer.Document)
	solrQ := &SolrQuery{queryChain: qchain, idxName: idx, docs: d}

	_, err := solrQ.execute()
	if err != nil {
		return nil, err
	}
	qresults := solrQ.results()
	objs := getResults(idx, qresults)
	res := make([]map[string]interface{}, len(objs))
	for i, r := range objs {
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

	/* If we're doing partial search, tease out the fields we want. */
	if partialData != nil {
		res, err = formatPartials(res, objs, partialData)
		if err != nil {
			return nil, err
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
	sortResults := results{res, sortKey}
	if ordering == "desc" {
		sort.Sort(sort.Reverse(sortResults))
	} else {
		sort.Sort(sortResults)
	}
	res = sortResults.res

	end := start + rows
	if end > len(res) {
		end = len(res)
	}
	res = res[start:end]
	return res, nil
}

func (sq *SolrQuery) execute() (map[string]indexer.Document, error) {
	s := sq.queryChain
	curOp := OpNotAnOp
	for s != nil {
		var r map[string]indexer.Document
		var err error
		switch c := s.(type) {
		case *SubQuery:
			_ = c
			newq, nend, nerr := extractSubQuery(s)
			if nerr != nil {
				return nil, err
			}
			s = nend
			var d map[string]indexer.Document
			if curOp == OpBinAnd {
				d = sq.docs
			} else {
				d = make(map[string]indexer.Document)
			}
			nsq := &SolrQuery{queryChain: newq, idxName: sq.idxName, docs: d}
			r, err = nsq.execute()
		default:
			if curOp == OpBinAnd {
				r, err = s.SearchResults(sq.docs)
			} else {
				r, err = s.SearchIndex(sq.idxName)
			}
		}
		if err != nil {
			return nil, err
		}
		if len(sq.docs) == 0 || curOp == OpBinAnd { // nothing in place yet
			sq.docs = r
		} else if curOp == OpBinOr {
			for k, v := range r {
				sq.docs[k] = v
			}
		} else {
			logger.Debugf("Somehow we got to what should have been an impossible state with search")
		}

		curOp = s.Op()
		s = s.Next()
	}
	return sq.docs, nil
}

func extractSubQuery(s Queryable) (Queryable, Queryable, error) {
	n := 1
	prev := s
	s = s.Next()
	top := s
	for {
		switch q := s.(type) {
		case *SubQuery:
			if q.start {
				n++
			} else {
				n--
			}
		}
		if n == 0 {
			// we've followed this subquery chain to its end
			prev.SetNext(nil) // snip this chain off at the end
			return top, s, nil
		}
		prev = s
		s = s.Next()
		if s == nil {
			break
		}
	}
	err := fmt.Errorf("Yikes! Somehow we weren't able to finish the subquery.")
	return nil, nil, err
}

func (sq *SolrQuery) results() []string {
	results := make([]string, len(sq.docs))
	n := 0
	for k := range sq.docs {
		results[n] = k
		n++
	}
	return results
}

// GetEndpoints gets a list from the indexer of all the endpoints available to
// search, namely the defaults (node, role, client, environment) and any data
// bags.
func (t *TrieSearch) GetEndpoints() []string {
	// TODO: deal with possible errors
	endpoints, _ := indexer.Endpoints()
	return endpoints
}

func getResults(variety string, toGet []string) []indexer.Indexable {
	var results []indexer.Indexable
	switch variety {
	case "node":
		ns, _ := node.GetMulti(toGet)
		// ....
		results = make([]indexer.Indexable, 0, len(ns))
		for _, n := range ns {
			results = append(results, n)
		}
	case "role":
		for _, r := range toGet {
			if role, _ := role.Get(r); role != nil {
				results = append(results, role)
			}
		}
	case "client":
		for _, c := range toGet {
			if client, _ := client.Get(c); client != nil {
				results = append(results, client)
			}
		}
	case "environment":
		for _, e := range toGet {
			if environment, _ := environment.Get(e); environment != nil {
				results = append(results, environment)
			}
		}
	default: // It's a data bag
		/* These may require further processing later. */
		dbag, _ := databag.Get(variety)
		if dbag != nil {
			for _, k := range toGet {
				dbi, err := dbag.GetDBItem(k)
				if err != nil {
					// at least log the error for
					// now
					logger.Errorf(err.Error())
				}
				results = append(results, dbi)
			}
		}
	}
	return results
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

func formatPartials(results []map[string]interface{}, objs []indexer.Indexable, partialData map[string]interface{}) ([]map[string]interface{}, error) {
	var err error
	results, err = partialSearchFormat(results, partialData)
	if err != nil {
		return nil, err
	}
	for x, z := range results {
		tmpRes := make(map[string]interface{})
		switch ro := objs[x].(type) {
		case *databag.DataBagItem:
			dbiURL := fmt.Sprintf("/data/%s/%s", ro.DataBagName, ro.RawData["id"].(string))
			tmpRes["url"] = util.CustomURL(dbiURL)
		default:
			tmpRes["url"] = util.ObjURL(objs[x].(util.GoiardiObj))
		}
		tmpRes["data"] = z

		results[x] = tmpRes
	}
	return results, nil
}
