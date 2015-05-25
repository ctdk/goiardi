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

package search

import (
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/indexer"
	//"github.com/ctdk/goiardi/util"
	"regexp"
)

type PostgresSearch struct {
}

type PgQuery struct {
	queryChain Queryable
	paths []string
	queryStrs []string
	arguments []string
}

func (p *PostgresSearch) Search(idx string, q string, rows int, sortOrder string, start int, partialData map[string]interface{}) ([]map[string]interface{}, error) {
	// keep up with the ersatz solr.
	qq := &Tokenizer{Buffer: q}
	qq.Init()
	if err := qq.Parse(); err != nil {
		return nil, err
	}
	qq.Execute()
	qchain := qq.Evaluate()

	pgQ := &PgQuery{ queryChain: qchain }

	logger.Debugf("what on earth is the chain? %q", qchain)
	err := pgQ.execute()
	if err != nil {
		return nil, err
	}

	// dummy
	dres := make([]map[string]interface{}, 0)
	return dres, nil
}

func (p *PostgresSearch) GetEndpoints() []string {
	// TODO: deal with possible errors
	endpoints, err := indexer.Endpoints()
	return endpoints
	if err != nil {
		panic(err)
	}
	return endpoints
}

func (pq *PgQuery) execute(startTableID ...*int) error {
	p := pq.queryChain
	//curOp := OpNotAnOp
	opMap := map[Op]string{
		OpNotAnOp: "(not an op)",
		OpUnaryNot: "not",
		OpUnaryReq: "req",
		OpUnaryPro: "pro",
		OpBinAnd: "and",
		OpBinOr: "or",
		OpBoost: "boost",
		OpFuzzy: "fuzzy",
		OpStartGroup: "start group",
		OpEndGroup: "end group",
		OpStartIncl: "start inc",
		OpEndIncl: "end inc",
		OpStartExcl: "start exc",
		OpEndExcl: "end exc",
	}
	var t *int
	if len(startTableID) == 0 {
		z := 0
		t = &z
	} else {
		t = startTableID[0]
	}
	for p != nil {
		switch c := p.(type) {
		case *BasicQuery:
			pq.paths = append(pq.paths, string(c.field))
			logger.Debugf("basic t%d: field: %s op: %s term: %+v complete %v", *t, c.field, opMap[c.op], c.term, c.complete)
			args, qstr := buildBasicQuery(c.field, c.term, t)
			pq.args = append(pq.args, args...)
			pq.queryStrs = append(pq.queryStrs, qstr)
			*t++
		case *GroupedQuery:
			pq.paths = append(pq.paths, string(c.field))
			logger.Debugf("grouped t%d: field: %s op: %s terms: %+v complete %v", *t, c.field, opMap[c.op], c.terms, c.complete)
			*t++
		case *RangeQuery:
			pq.paths = append(pq.paths, string(c.field))
			logger.Debugf("range t%d: field %s op %s start %s end %s inclusive %v complete %v", *t, c.field, opMap[c.op], c.start, c.end, c.inclusive, c.complete)
			*t++
		case *SubQuery:
			logger.Debugf("STARTING SUBQUERY: op %s complete %v", opMap[c.op], c.complete)
			newq, nend, nerr := extractSubQuery(c)
			if nerr != nil {
				return nerr
			}
			p = nend
			logger.Debugf("OP NOW: %s", opMap[p.Op()])
			np := &PgQuery{ queryChain: newq }
			err := np.execute(t)
			if err != nil {
				return err
			}
			pq.paths = append(pq.paths, np.paths...)
			logger.Debugf("ENDING SUBQUERY")
		default:
			err := fmt.Errorf("Unknown type %T for query", c)
			return err
		}
		//curOp = p.Op()
		p = p.Next()
	}
	logger.Debugf("paths: %v", pq.paths)
	logger.Debugf("number of tables: %d", *t)
	return nil
}

func buildBasicQuery(field Field, term Term, tNum *int) ([]string, string) {
	var op string
	r := regex.MustCompile(`\*|\?`)
	if r.MatchString(term) {
		op = "LIKE"
	} else {
		op = "="
	}
	var q string
	args := []string{ field }
	if term == "*" {
		q = fmt.Sprintf("f%d.path ~ _ARG_")
	} else {
		q = fmt.Sprintf("f%d.path ~ _ARG_ AND f%d.value %s _ARG_", tNum, tNum, op)
		args = append(args, term)
	}

	return args, q
}
