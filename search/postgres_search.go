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

package search

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
)

type PostgresSearch struct {
}

type PgQuery struct {
	idx        string
	queryChain Queryable
	paths      []string
	queryStrs  []string
	arguments  []string
	fullQuery  string
	allArgs    []interface{}
	searchSchema string
}

type gClause struct {
	clause string
	op     Op
}

func (p *PostgresSearch) Search(org *organization.Organization, idx string, q string, rows int, sortOrder string, start int, partialData map[string]interface{}) ([]map[string]interface{}, error) {
	// get the search schema name
	searchSchema := org.SearchSchemaName()

	// check that the endpoint actually exists
	sqlStmt := fmt.Sprintf("SELECT 1 FROM %s.search_collections WHERE organization_id = $1 AND name = $2", searchSchema)
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

	// Don't start timing searches until the existence of the index has
	// been checked.
	defer trackSearchTiming(time.Now(), q, pgSearchTimings)

	// Special case "goodness". If the search term is "*:*" with no
	// qualifiers, short circuit everything and just get a list of the
	// distinct items.
	var qresults []string

	if q == "*:*" {
		searchQueryDebugf("Searching '*:*' on %s, short circuiting", idx)

		var builtinIdx bool
		if idx == "node" || idx == "client" || idx == "environment" || idx == "role" {
			builtinIdx = true
			sqlStmt = fmt.Sprintf("SELECT COALESCE(ARRAY_AGG(name), '{}'::text[]) FROM %s.%ss WHERE organization_id = $1", searchSchema, idx)
		} else {
			sqlStmt = fmt.Sprintf("SELECT COALESCE(ARRAY_AGG(orig_name), '{}'::text[]) FROM %s.data_bag_items JOIN %s.data_bags ON %s.data_bag_items.data_bag_id = %s.data_bags.id WHERE %s.data_bags.organization_id = $1 AND %s.data_bags.name = $2", searchSchema, searchSchema, searchSchema, searchSchema, searchSchema, searchSchema)
		}

		var res util.StringSlice
		stmt, err := datastore.Dbh.Prepare(sqlStmt)
		if err != nil {
			return nil, err
		}
		defer stmt.Close()
		if builtinIdx {
			err = stmt.QueryRow(1).Scan(&res)
		} else {
			err = stmt.QueryRow(1, idx).Scan(&res)
		}
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		qresults = res
	} else {
		// keep up with the ersatz solr.
		qq := &Tokenizer{Buffer: q}
		qq.Init()
		if err := qq.Parse(); err != nil {
			return nil, err
		}
		qq.Execute()
		qchain := qq.Evaluate()

		pgQ := &PgQuery{idx: idx, queryChain: qchain, searchSchema: searchSchema}

		err := pgQ.execute()
		if err != nil {
			return nil, err
		}

		qresults, err = pgQ.results()
		if err != nil {
			return nil, err
		}
	}
	// THE WRONG WAY:
	// Eventually, ordering by the keys themselves would be awesome.
	objs := getResults(org, idx, qresults)
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
		var err error
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

func (p *PostgresSearch) GetEndpoints(org *organization.Organization) []string {
	// TODO: deal with possible errors
	endpoints, err := indexer.Endpoints(org.Name)
	if err != nil {
		logger.Errorf(err.Error())
	}
	return endpoints
}

func (pq *PgQuery) execute(startTableID ...*int) error {
	p := pq.queryChain
	curOp := OpNotAnOp
	var t *int
	if len(startTableID) == 0 {
		z := 0
		t = &z
	} else {
		t = startTableID[0]
	}
	for p != nil {
		searchQueryDebugf("p: %T op %s %+v", p, opMap[p.Op()], p)
		searchQueryDebugf("curOp: %s", opMap[curOp])
		switch c := p.(type) {
		case *BasicQuery:
			// an empty field can only happen up here
			if c.field != "" {
				pq.paths = append(pq.paths, string(c.field))
			}
			args, xtraPath, qstr := buildBasicQuery(c.field, c.term, t, curOp, pq.searchSchema)
			if xtraPath != "" {
				pq.paths = append(pq.paths, xtraPath)
			}
			pq.arguments = append(pq.arguments, args...)
			pq.queryStrs = append(pq.queryStrs, qstr)
			*t++
		case *GroupedQuery:
			pq.paths = append(pq.paths, string(c.field))
			args, xtraPath, qstr := buildGroupedQuery(c.field, c.terms, t, curOp, pq.searchSchema)
			if xtraPath != "" {
				pq.paths = append(pq.paths, xtraPath)
			}
			pq.arguments = append(pq.arguments, args...)
			pq.queryStrs = append(pq.queryStrs, qstr)
			*t++
		case *RangeQuery:
			pq.paths = append(pq.paths, string(c.field))
			args, xtraPath, qstr := buildRangeQuery(c.field, c.start, c.end, c.inclusive, t, curOp, pq.searchSchema, p.(*RangeQuery).negated)
			if xtraPath != "" {
				pq.paths = append(pq.paths, xtraPath)
			}
			pq.arguments = append(pq.arguments, args...)
			pq.queryStrs = append(pq.queryStrs, qstr)
			*t++
		case *SubQuery:
			newq, nend, nerr := extractSubQuery(c)
			if nerr != nil {
				return nerr
			}
			p = nend
			np := &PgQuery{queryChain: newq, searchSchema: pq.searchSchema}
			err := np.execute(t)
			if err != nil {
				return err
			}
			pq.paths = append(pq.paths, np.paths...)
			pq.arguments = append(pq.arguments, np.arguments...)
			pq.queryStrs = append(pq.queryStrs, fmt.Sprintf("%s(%s)", binOp(curOp), strings.Join(np.queryStrs, " ")))
		case *NotQuery: // skip to the next thing...?
			// act to preserve the previous query's op (if there was
			// one.
			p.AddOp(curOp)
		default:
			err := fmt.Errorf("Unknown type %T for query", c)
			return err
		}
		curOp = p.Op()
		p = p.Next()
	}
	fullQ, allArgs := craftFullQuery(1, pq.idx, pq.paths, pq.arguments, pq.queryStrs, t, pq.searchSchema)
	searchQueryDebugf("pg search info:")
	searchQueryDebugf("pg search schema: %s", pq.searchSchema)
	searchQueryDebugf("full query: %s", fullQ)
	searchQueryDebugf("all %d args: %v", len(allArgs), allArgs)
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
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return res, nil
}

func buildBasicQuery(field Field, term QueryTerm, tNum *int, op Op, searchSchema string) ([]string, string, string) {
	opStr := binOp(op)
	originalTerm := term.term
	cop := matchOp(term.mod, &term)

	var q string
	args := []string{string(field)}
	var xtraPath string
	if originalTerm == "*" || originalTerm == "" {
		q = fmt.Sprintf("%s(f%d.path ~ _ARG_)", opStr, *tNum)
	} else if field == "" { // feeling REALLY iffy about this one, but it
		// duplicates the previous behavior.
		q = fmt.Sprintf("%s(f%d.value %s _ARG_)", opStr, *tNum, cop)
		args = []string{string(term.term)}
	} else {
		// For ltree, change this *back*.
		// Strictly speaking, certain kinds of query won't have exactly
		// the same behavior as you would get with solr, but it only
		// comes up in a few corner cases that should be unlikely in
		// real world searching. (Famous last words.) It should only be
		// queries like "foo:bar*" or "foo:bar?" where "foo.bar*" is a
		// ltree path rather than a path and value, because searches
		// with ? matching single characters won't work right, and
		// wildcard searches with * might not behave quite the way one
		// expects (*maybe*). In practice it shouldn't be a huge
		// problem.
		altQueryPath := craftAltQueryPath(string(field), string(originalTerm))
		var clauseJoin string
		var notPath bool
		if term.mod == OpUnaryNot || term.mod == OpUnaryPro {
			clauseJoin = "AND NOT"
			notPath = true
		} else {
			clauseJoin = "OR"
		}
		// the extra part for negated queries:
		// (<part below> OR NOT EXISTS (SELECT 1 FROM found_items WHERE
		// i.item_name = found_items.item_name AND found_items.path
		// OPERATOR(goiardi.~) 'action'))

		q = fmt.Sprintf("((f%d.path OPERATOR(%s.~) _ARG_ AND f%d.value %s _ARG_) %s (f%d.path OPERATOR(%s.~) _ARG_))", *tNum, searchSchema, *tNum, cop, clauseJoin, *tNum, searchSchema)
		if notPath {
			q = fmt.Sprintf("(" + q + " OR NOT EXISTS (SELECT 1 FROM found_items WHERE i.item_name = found_items.item_name AND found_items.path OPERATOR(%s.~) _ARG_))", searchSchema)
		}
		/*******
		q = fmt.Sprintf("%s((f%d.path OPERATOR(%s.~) _ARG_ AND f%d.value %s _ARG_) %s (f%d.path OPERATOR(%s.~) _ARG_))", opStr, *tNum, searchSchema, *tNum, cop, clauseJoin, *tNum, searchSchema)
		*******/
		q = opStr + q

		args = append(args, string(term.term))
		args = append(args, altQueryPath)

		if notPath {
			args = append(args, string(field))
		}

		xtraPath = altQueryPath
	}

	return args, xtraPath, q
}

func buildGroupedQuery(field Field, terms []QueryTerm, tNum *int, op Op, searchSchema string) ([]string, string, string) {
	opStr := binOp(op)

	var q string
	args := []string{string(field)}
	var grouped []*gClause

	basePath := string(field)
	xtraPath := fmt.Sprintf("%s.*", string(field))
	var groupedPaths []*gClause
	var groupedArgs []string
	ltNum := *tNum

	for _, v := range terms {
		orgTerm := v.term
		cop := matchOp(op, &v)

		clause := fmt.Sprintf("f%d.value %s _ARG_", *tNum, cop)
		g := &gClause{clause, v.mod}
		grouped = append(grouped, g)

		var ltreeNot string
		if v.mod == OpUnaryNot {
			ltreeNot = "!"
		}
		groupedArgs = append(groupedArgs, fmt.Sprintf("%s.%s%s", basePath, ltreeNot, util.PgSearchQueryKey(string(orgTerm))))
		ltNum++

		groupedPaths = append(groupedPaths, &gClause{fmt.Sprintf("f%d.path OPERATOR(%s.~) _ARG_", ltNum, searchSchema), v.mod})
		args = append(args, string(v.term))
	}
	var clauseArr []string
	var ltClauseArr []string
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
	for i, lc := range groupedPaths {
		var j string
		if i != 0 {
			if lc.op == OpUnaryPro || lc.op == OpUnaryReq || lc.op == OpUnaryNot {
				j = " AND "
			} else {
				j = " OR "
			}
		}
		ltClauseArr = append(ltClauseArr, fmt.Sprintf("%s%s", j, lc.clause))
	}
	clauses := strings.Join(clauseArr, " ")
	ltClauses := strings.Join(ltClauseArr, " ")
	q = fmt.Sprintf("%s((f%d.path OPERATOR(%s.~) _ARG_ AND (%s)) OR (%s))", opStr, *tNum, searchSchema, clauses, ltClauses)
	*tNum = ltNum
	args = append(args, groupedArgs...)
	return args, xtraPath, q
}

func buildRangeQuery(field Field, start RangeTerm, end RangeTerm, inclusive bool, tNum *int, op Op, searchSchema string, negated bool) ([]string, string, string) {
	if start > end {
		start, end = end, start
	}

	var q string
	args := []string{string(field)}
	xtraPath := fmt.Sprintf("%s.*", string(field))

	opStr := binOp(op)

	var notRange string
	if negated {
		notRange = " NOT "
	}

	var equals string
	if inclusive {
		equals = "="
	}
	var ranges []string
	var rangePaths []string
	var rangeArgs []string // these need to be added to the args after

	if string(start) != "*" {
		s := fmt.Sprintf("f%d.value >%s _ARG_", *tNum, equals)
		ranges = append(ranges, s)
		args = append(args, string(start))
		rangePaths = append(rangePaths, fmt.Sprintf("f%d.path OPERATOR(%s.>%s) _ARG_", *tNum, searchSchema, equals))
		rangeArgs = append(rangeArgs, fmt.Sprintf("%s.%s", string(field), util.PgSearchQueryKey(string(start))))
	}
	if string(end) != "*" {
		e := fmt.Sprintf("f%d.value <%s _ARG_", *tNum, equals)
		ranges = append(ranges, e)
		args = append(args, string(end))
		rangePaths = append(rangePaths, fmt.Sprintf("f%d.path OPERATOR(%s.<%s) _ARG_", *tNum, searchSchema, equals))
		rangeArgs = append(rangeArgs, fmt.Sprintf("%s.%s", string(field), util.PgSearchQueryKey(string(end))))
	}

	args = append(args, xtraPath)
	if len(rangeArgs) != 0 {
		args = append(args, rangeArgs...)
	}

	var rangeStr string
	var rangePathStr string
	if len(ranges) != 0 {
		rangeStr = fmt.Sprintf(" AND (%s)", strings.Join(ranges, " AND "))
		// Add the first path match to keep the range query with ltrees
		// from shooting off into the distance
		rangePathStr = fmt.Sprintf(" OR (f%d.path OPERATOR(%s.~) _ARG_ AND %s)", *tNum, searchSchema, strings.Join(rangePaths, " AND "))
	}
	q = fmt.Sprintf("%s%s(f%d.path OPERATOR(%s.~) _ARG_%s%s)", opStr, notRange, *tNum, searchSchema, rangeStr, rangePathStr)
	return args, xtraPath, q
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

func craftFullQuery(orgID int, idx string, paths []string, arguments []string, queryStrs []string, tNum *int, searchSchema string) (string, []interface{}) {
	allArgs := make([]interface{}, 0, len(paths)+len(arguments)+2)
	allArgs = append(allArgs, orgID)
	allArgs = append(allArgs, idx)

	pcount := 3

	for _, v := range paths {
		allArgs = append(allArgs, v)
	}
	for _, v := range arguments {
		allArgs = append(allArgs, v)
	}

	var itemsStatement string
	if idx == "node" || idx == "client" || idx == "environment" || idx == "role" {
		itemsStatement = fmt.Sprintf("SELECT name AS item_name FROM %s.%ss WHERE organization_id = $1", searchSchema, idx)
	} else {
		itemsStatement = fmt.Sprintf("SELECT orig_name AS item_name FROM %s.data_bag_items JOIN %s.data_bags ON %s.data_bag_items.data_bag_id = %s.data_bags.id WHERE %s.data_bags.organization_id = $1 AND %s.data_bags.name = $2", searchSchema, searchSchema, searchSchema, searchSchema, searchSchema, searchSchema)
		pcount = 3
	}

	params := make([]string, 0, len(paths))
	for range paths {
		params = append(params, fmt.Sprintf("$%d", pcount))
		pcount++
	}

	withStatement := fmt.Sprintf("WITH found_items AS (SELECT item_name, path, value FROM %s.search_items si WHERE si.organization_id = $1 AND si.search_collection_id = (SELECT id FROM %s.search_collections WHERE name = $2) AND path OPERATOR(%s.?) ARRAY[ %s ]::%s.lquery[]), items AS (%s)", searchSchema, searchSchema, searchSchema, strings.Join(params, ", "), searchSchema, itemsStatement)
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
	fullQuery := strings.Join([]string{withStatement, selectStmt}, " ")
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

func craftAltQueryPath(a, b string) string {
	return util.PgSearchQueryKey(strings.Join([]string{a, b}, "."))
}

func searchQueryDebugf(format string, args ...interface{}) {
	if config.Config.SearchQueryDebug {
		logger.Debugf(format, args...)
	}
}
