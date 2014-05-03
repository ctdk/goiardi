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
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/data_bag"
	"net/url"
	"fmt"
	"git.tideland.biz/goas/logger"
)

// Holds a parsed query and query chain to run against the index.
type SolrQuery struct {
	queryChain Queryable
	idxName string
	docs map[string]*indexer.IdxDoc
}

// Parse the given query string and search the given index for any matching
// results.
func Search(idx string, q string) ([]indexer.Indexable, error) {
	/* Eventually we'll want more prep. To start, look right in the index */
	query, qerr := url.QueryUnescape(q)
	if qerr != nil {
		return nil, qerr
	}
	qq := &Tokenizer{ Buffer: query }
	qq.Init()
	if err := qq.Parse(); err != nil {
		return nil, err
	}
	qq.Execute()
	qchain := qq.Evaluate()
	d := make(map[string]*indexer.IdxDoc)
	solrQ := &SolrQuery{ queryChain: qchain, idxName: idx, docs: d, }

	_, err := solrQ.execute()
	if err != nil {
		return nil, err
	}
	results := solrQ.results()
	objs := getResults(idx, results)
	return objs, nil
}

func (sq *SolrQuery) execute() (map[string]*indexer.IdxDoc, error) {
	s := sq.queryChain
	curOp := OpNotAnOp
	for s != nil {
		var r map[string]*indexer.IdxDoc
		var err error
		switch c := s.(type){
			case *SubQuery:
				_ = c
				newq, nend, nerr := extractSubQuery(s)
				if nerr != nil {
					return nil, err
				}
				s = nend
				d := make(map[string]*indexer.IdxDoc)
				nsq := &SolrQuery{ queryChain: newq, idxName: sq.idxName, docs: d }
				r, err = nsq.execute()
			default:
				r, err = s.SearchIndex(sq.idxName)
		}
		if err != nil {
			return nil, err
		}
		if len(sq.docs) == 0 { // nothing in place yet
			sq.docs = r
		} else if curOp == OpBinOr {
			for k, v := range r {
				sq.docs[k] = v
			}
		} else if curOp == OpBinAnd {
			newRes := make(map[string]*indexer.IdxDoc, len(sq.docs) + len(r))
			for k, v := range sq.docs {
				if _, found := r[k]; found {
					newRes[k] = v
				}
			}
			sq.docs = newRes
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

func (sq *SolrQuery) results() ([]string) {
	results := make([]string, len(sq.docs))
	n := 0
	for k := range sq.docs {
		results[n] = k
		n++
	}
	return results
}

// Get a list from the indexer of all the endpoints available to search.
func GetEndpoints() []string {
	endpoints := indexer.Endpoints()
	return endpoints
}

func getResults(variety string, toGet []string) []indexer.Indexable {
	results := make([]indexer.Indexable, 0)
	switch variety {
		case "node":
			for _, n := range toGet {
				if node, _ := node.Get(n); node != nil {
					results = append(results, node)
				}
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
			dbag, _ := data_bag.Get(variety)
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
