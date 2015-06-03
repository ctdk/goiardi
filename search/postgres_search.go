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
	"database/sql"
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
	"regexp"
	"sort"
	"strings"
	"github.com/ctdk/goiardi/client"
)

type PostgresSearch struct {
}

type PgQuery struct {
	idx string
	queryChain Queryable
	paths []string
	queryStrs []string
	arguments []string
	fullQuery string
	allArgs []interface{}
}

type gClause struct {
	clause string
	op Op
}

func (p *PostgresSearch) Search(idx string, q string, rows int, sortOrder string, start int, partialData map[string]interface{}) ([]map[string]interface{}, error) {
	// check that the endpoint actually exists
	sqlStmt := "SELECT 1 FROM goiardi.search_collections WHERE organization_id = $1 AND name = $2"
	stmt, serr := datastore.Dbh.Prepare(sqlStmt)
	if serr != nil {
		return nil, serr
	}
	defer stmt.Close()
	var zzz int
	serr = stmt.QueryRow(1, idx).Scan(&zzz) // don't care about zzz
	if serr != nil {
		if serr == sql.ErrNoRows {
			serr = fmt.Errorf("I don't know how to search for %s data objects.", idx)
		}
		return nil, serr
	}

	// keep up with the ersatz solr.
	qq := &Tokenizer{Buffer: q}
	qq.Init()
	if err := qq.Parse(); err != nil {
		return nil, err
	}
	qq.Execute()
	qchain := qq.Evaluate()

	pgQ := &PgQuery{ idx: idx, queryChain: qchain }

	logger.Debugf("what on earth is the chain? %q", qchain)
	err := pgQ.execute()
	if err != nil {
		return nil, err
	}

	qresults, err := pgQ.results()
	if err != nil {
		return nil, err
	}
	// THE WRONG WAY:
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
	curOp := OpNotAnOp
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
			// an empty field can only happen up here
			if c.field != "" {
				pq.paths = append(pq.paths, string(c.field))
			}
			logger.Debugf("basic t%d: field: %s op: %s term: %+v complete %v", *t, c.field, opMap[c.op], c.term, c.complete)
			args, qstr := buildBasicQuery(c.field, c.term, t, curOp)
			logger.Debugf("qstr: %s", qstr)
			pq.arguments = append(pq.arguments, args...)
			pq.queryStrs = append(pq.queryStrs, qstr)
			*t++
		case *GroupedQuery:
			pq.paths = append(pq.paths, string(c.field))
			logger.Debugf("grouped t%d: field: %s op: %s terms: %+v complete %v", *t, c.field, opMap[c.op], c.terms, c.complete)
			args, qstr := buildGroupedQuery(c.field, c.terms, t, curOp)
			logger.Debugf("qstr: %s", qstr)
			pq.arguments = append(pq.arguments, args...)
			pq.queryStrs = append(pq.queryStrs, qstr)
			*t++
		case *RangeQuery:
			pq.paths = append(pq.paths, string(c.field))
			logger.Debugf("range t%d: field %s op %s start %s end %s inclusive %v complete %v", *t, c.field, opMap[c.op], c.start, c.end, c.inclusive, c.complete)
			args, qstr := buildRangeQuery(c.field, c.start, c.end, c.inclusive, t, curOp)
			logger.Debugf("qstr: %s", qstr)
			pq.arguments = append(pq.arguments, args...)
			pq.queryStrs = append(pq.queryStrs, qstr)
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
			logger.Debugf("subquery paths: %v", np.paths)
			logger.Debugf("subquery args: %v", np.arguments)
			logger.Debugf("subquery qstrs: %v", np.queryStrs)
			pq.paths = append(pq.paths, np.paths...)
			pq.arguments = append(pq.arguments, np.arguments...)
			pq.queryStrs = append(pq.queryStrs, fmt.Sprintf("%s(%s)", binOp(curOp), strings.Join(np.queryStrs, " ")))
			logger.Debugf("ENDING SUBQUERY")
		default:
			err := fmt.Errorf("Unknown type %T for query", c)
			return err
		}
		curOp = p.Op()
		p = p.Next()
	}
	logger.Debugf("paths: %v", pq.paths)
	logger.Debugf("arguments: %v", pq.arguments)
	logger.Debugf("query strings: %v", pq.queryStrs)
	logger.Debugf("number of tables: %d", *t)
	fullQ, allArgs := craftFullQuery(1, pq.idx, pq.paths, pq.arguments, pq.queryStrs, t)
	logger.Debugf("full query: %s", fullQ)
	logger.Debugf("all %d args: %v", len(allArgs), allArgs)
	pq.fullQuery = fullQ
	pq.allArgs = allArgs
	return nil
}

func (pq *PgQuery) results() ([]string, error) {
	var res util.StringSlice
	stmt, err := datastore.Dbh.Prepare(pq.fullQuery)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(pq.allArgs...).Scan(&res)
	if err != nil && err != sql.ErrNoRows{
		return nil, err
	}
	logger.Debugf("res? %v", res)
	return res, nil
}

func buildBasicQuery(field Field, term QueryTerm, tNum *int, op Op) ([]string, string) {
	opStr := binOp(op)
	originalTerm := term.term
	cop := matchOp(term.mod, &term)

	var q string
	args := []string{ string(field) }
	if originalTerm == "*" || originalTerm == "" {
		q = fmt.Sprintf("%s(f%d.path ~ _ARG_)", opStr, *tNum)
	} else if field == "" { // feeling REALLY iffy about this one, but it
				// duplicates the previous behavior.
		q = fmt.Sprintf("%s(f%d.value %s _ARG_)", opStr, *tNum, cop)
		args = []string{ string(term.term) }
	} else {
		q = fmt.Sprintf("%s(f%d.path ~ _ARG_ AND f%d.value %s _ARG_)", opStr, *tNum, *tNum, cop)
		args = append(args, string(term.term))
	}

	return args, q
}

func buildGroupedQuery(field Field, terms []QueryTerm, tNum *int, op Op) ([]string, string) {
	opStr := binOp(op)

	var q string
	args := []string{ string(field) }
	var grouped []*gClause

	for _, v := range terms {
		cop := matchOp(op, &v)
		
		clause := fmt.Sprintf("f%d.value %s _ARG_", *tNum, cop)
		g := &gClause{ clause, v.mod }
		grouped = append(grouped, g)
		args = append(args, string(v.term))
	}
	var clauseArr []string
	for i, g := range grouped {
		var j string
		if i != 0 {
			if g.op == OpUnaryPro || g.op == OpUnaryReq || g.op == OpUnaryNot {
				j = " AND "
			} else {
				j = " OR "
			}
		}
		clauseArr = append(clauseArr, fmt.Sprintf("%s%s", j, g.clause))
	}
	clauses := strings.Join(clauseArr, " ")
	q = fmt.Sprintf("%s(f%d.path ~ _ARG_ AND (%s))", opStr, *tNum, clauses)
	return args, q
}

func buildRangeQuery(field Field, start RangeTerm, end RangeTerm, inclusive bool, tNum *int, op Op) ([]string, string) {
	if start > end {
		start, end = end, start
	}

	var q string
	args := []string{ string(field) }

	opStr := binOp(op)
	var equals string
	if inclusive {
		equals = "="
	}
	var ranges []string
	if string(start) != "*" {
		s := fmt.Sprintf("f%d.value >%s _ARG_", *tNum, equals)
		ranges = append(ranges, s)
		args = append(args, string(start))
	}
	if string(end) != "*" {
		e := fmt.Sprintf("f%d.value <%s _ARG_", *tNum, equals)
		ranges = append(ranges, e)
		args = append(args, string(end))
	}
	var rangeStr string
	if len(ranges) != 0 {
		rangeStr = fmt.Sprintf(" AND (%s)", strings.Join(ranges, " AND "))
	}
	q = fmt.Sprintf("%s(f%d.path ~ _ARG_%s)", opStr, *tNum, rangeStr)
	return args, q
}

func matchOp(op Op, term *QueryTerm) string {
	r := regexp.MustCompile(`\*|\?`)
	var cop string
	if r.MatchString(string(term.term)) {
		if term.mod == OpUnaryNot || term.mod == OpUnaryPro {
			cop = "NOT LIKE"
		} else {
			cop = "LIKE"
		}
		term.term = Term(escapeArg(string(term.term)))
	} else {
		if term.mod == OpUnaryNot || term.mod == OpUnaryPro {
			cop = "<>"
		} else {
			cop = "="
		}
	}
	return cop
}

func binOp(op Op) string {
	var opStr string
	if op != OpNotAnOp {
		if op == OpBinAnd {
			opStr = " AND "
		} else {
			opStr = " OR "
		}
	}
	return opStr
}

func craftFullQuery(orgID int, idx string, paths []string, arguments []string, queryStrs []string, tNum *int) (string, []interface{}) {
	// TODO: FIX
	allArgs := make([]interface{}, 0, len(paths) + len(arguments) + 2)
	allArgs = append(allArgs, orgID)
	allArgs = append(allArgs, idx)
	for _, v := range paths {
		allArgs = append(allArgs, v)
	}
	for _, v := range arguments {
		allArgs = append(allArgs, v)
	}

	pcount := 3
	params := make([]string, 0, len(paths))
	for range paths {
		params = append(params, fmt.Sprintf("$%d", pcount))
		pcount++
	}
	withStatement := fmt.Sprintf("WITH found_items AS (SELECT item_name, path, value FROM goiardi.search_items si JOIN goiardi.search_collections sc ON si.search_collection_id = sc.id WHERE si.organization_id = $1 AND sc.name = $2 AND path ? ARRAY[ %s ]::goiardi.lquery[]), items AS (SELECT DISTINCT item_name FROM found_items)", strings.Join(params, ", "))
	var selectStmt string
	if *tNum == 1 {
		selectStmt = fmt.Sprintf("SELECT COALESCE(ARRAY_AGG(DISTINCT item_name), '{}'::text[]) FROM found_items f0 WHERE %s", queryStrs[0])
	} else {
		joins := make([]string, 0, *tNum)
		for i := 0; i < *tNum; i++ {
			j := fmt.Sprintf("INNER JOIN found_items AS f%d ON i.item_name = f%d.item_name", i, i)
			joins = append(joins, j)
		}
		selectStmt = fmt.Sprintf("SELECT COALESCE(ARRAY_AGG(i.item_name), '{}'::text[]) FROM items i %s WHERE %s", strings.Join(joins, " "), strings.Join(queryStrs, " "))
	}
	fullQuery := strings.Join([]string{ withStatement, selectStmt }, " ")
	re := regexp.MustCompile("_ARG_")
	rfunc := func([]byte) []byte {
		r := []byte(fmt.Sprintf("$%d", pcount))
		pcount++
		return r
	}
	fullQuery = string(re.ReplaceAllFunc([]byte(fullQuery), rfunc))

	return fullQuery, allArgs
}

func escapeArg(arg string) string {
	arg = strings.Replace(arg, "%", "\\%", -1)
	arg = strings.Replace(arg, "_", "\\_", -1)
	arg = strings.Replace(arg, "*", "%", -1)
	arg = strings.Replace(arg, "?", "_", -1)
	return arg
}
