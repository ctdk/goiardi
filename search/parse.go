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

//Parse Solr queries with the PEG generated from 'search/search-parse.peg',
//located in search-parse.peg.go. To have changes to seach-parse.peg reflected
//in search-parse.peg.go, install peg from https://github.com/pointlander/peg
//and run 'peg -switch -inline search-parse.peg'.

package search

import (
	"errors"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"strings"
)

// Op is a search operator
type Op uint8

// Field is a field in a document or object to search for, like when searching
// for clients with "field:*".
type Field string

// Term is a basic search term string.
type Term string

// RangeTerm is a string, but describes a range to search over, like 1-10.
type RangeTerm string

func (t Term) String() string {
	return string(t)
}

func (f Field) String() string {
	return string(f)
}

func (r RangeTerm) String() string {
	return string(r)
}

// Define the various search operations.
const (
	OpNotAnOp Op = iota
	OpUnaryNot
	OpUnaryReq
	OpUnaryPro
	OpBinAnd
	OpBinOr
	OpBoost
	OpFuzzy
	OpStartGroup
	OpEndGroup
	OpStartIncl
	OpEndIncl
	OpStartExcl
	OpEndExcl
)

var opMap = map[Op]string{
	OpNotAnOp:    "OpNotAnOp",
	OpUnaryNot:   "OpUnaryNot",
	OpUnaryReq:   "OpUnaryReq",
	OpUnaryPro:   "OpUnaryPro",
	OpBinAnd:     "OpBinAnd",
	OpBinOr:      "OpBinOr",
	OpBoost:      "OpBoost",
	OpFuzzy:      "OpFuzzy",
	OpStartGroup: "OpStartGroup",
	OpEndGroup:   "OpEndGroup",
	OpStartIncl:  "OpStartIncl",
	OpEndIncl:    "OpEndIncl",
	OpStartExcl:  "OpStartExcl",
	OpEndExcl:    "OpEndExcl",
}

// Token is a parsed token from the solr query.
type Token struct {
	QueryChain Queryable
	Latest     Queryable
}

// QueryTerm is an individual query term and its operator.
type QueryTerm struct {
	term      Term
	mod       Op
	fuzzboost Op
	fuzzparam string
}

// BasicQuery is the no frills basic query type, without groups or ranges.
// Can contain regexp terms, however.
type BasicQuery struct {
	field    Field
	term     QueryTerm
	op       Op
	next     Queryable
	prev     Queryable
	complete bool
}

// GroupedQuery is for a query with grouped results.
type GroupedQuery struct {
	field    Field
	terms    []QueryTerm
	op       Op
	next     Queryable
	prev     Queryable
	complete bool
}

// RangeQuery is for a Query a range of values.
type RangeQuery struct {
	field     Field
	start     RangeTerm
	end       RangeTerm
	inclusive bool
	op        Op
	next      Queryable
	prev      Queryable
	complete  bool
	negated   bool
}

// SubQuery is really just a marker in the chain of queries. Later it will be
// processed by itself though.
type SubQuery struct {
	start    bool
	end      bool
	op       Op
	complete bool
	next     Queryable
	prev     Queryable
}

type NotQuery struct {
	op   Op
	next Queryable
	prev Queryable
}

// Queryable defines an interface of methods all the query chain types have to
// be able to implement to search the index.
type Queryable interface {
	// Search the index for the given term.
	SearchIndex(*organization.Organization, string) (map[string]indexer.Document, error)
	// Search for the given term from already gathered search results
	SearchResults(map[string]indexer.Document) (map[string]indexer.Document, error)
	// Add an operator to this query chain link.
	AddOp(Op)
	// Get this query chain link's op.
	Op() Op
	// Add a field to this query chain link.
	AddField(Field)
	// Add a term to this link.
	AddTerm(Term)
	// Add an operator to the query chain link's term.
	AddTermOp(Op)
	// Set the next query in the query chain.
	SetNext(Queryable)
	// Set the previous query token in the query chain.
	SetPrev(Queryable)
	// Get the next link in the query chain.
	Next() Queryable
	// Get the previous link in the query chain.
	Prev() Queryable
	// Is the query chain incomplete?
	IsIncomplete() bool
	// Sets the completed flag for this query chain on this link.
	SetCompleted()
	// Add fuzz boost to the query. NOTE: doesn't do much.
	AddFuzzBoost(Op)
	// Add a fuzz param to the query. NOTE: doesn't do much.
	AddFuzzParam(string)
}

type groupQueryHolder struct {
	op  Op
	res map[string]indexer.Document
}

func (q *BasicQuery) SearchIndex(org *organization.Organization, idxName string) (map[string]indexer.Document, error) {
	notop := false
	if q.Prev() != nil && ((q.Prev().Op() == OpUnaryNot) || (q.Prev().Op() == OpUnaryPro)) {
		notop = true
	}
	i := indexer.GetIndex()
	if q.field == "" {
		res, err := i.SearchText(org, idxName, string(q.term.term), notop)
		return res, err
	}
	searchTerm := makeSearchTerm(q.field, q.term.term)
	res, err := i.Search(org, idxName, searchTerm, notop)

	return res, err
}

func (q *BasicQuery) SearchResults(curRes map[string]indexer.Document) (map[string]indexer.Document, error) {
	notop := false
	if q.Prev() != nil && ((q.Prev().Op() == OpUnaryNot) || (q.Prev().Op() == OpUnaryPro)) {
		notop = true
	}
	// TODO: add field == ""

	searchTerm := makeSearchTerm(q.field, q.term.term)
	i := indexer.GetIndex()
	res, err := i.SearchResults(searchTerm, notop, curRes)

	return res, err
}

func (q *BasicQuery) AddOp(o Op) {
	q.op = o
}

func (q *BasicQuery) Op() Op {
	return q.op
}

func (q *BasicQuery) AddField(s Field) {
	if config.Config.ConvertSearch {
		s = Field(util.PgSearchQueryKey(string(s)))
	}
	q.field = s
}

func (q *BasicQuery) AddTerm(s Term) {
	q.term.term = s
	if q.Prev() != nil && ((q.Prev().Op() == OpUnaryNot) || (q.Prev().Op() == OpUnaryPro)) {
		q.AddTermOp(q.Prev().Op())
	}

	q.SetCompleted()
}

func (q *BasicQuery) AddTermOp(o Op) {
	q.term.mod = o
}

func (q *BasicQuery) SetNext(n Queryable) {
	q.next = n
}

func (q *BasicQuery) Next() Queryable {
	return q.next
}

func (q *BasicQuery) SetPrev(n Queryable) {
	q.prev = n
}

func (q *BasicQuery) Prev() Queryable {
	return q.prev
}

func (q *BasicQuery) IsIncomplete() bool {
	return !q.complete
}

func (q *BasicQuery) SetCompleted() {
	q.complete = true
}
func (q *BasicQuery) AddFuzzBoost(o Op) {
	q.term.fuzzboost = o
}

func (q *BasicQuery) AddFuzzParam(s string) {
	q.term.fuzzparam = s
}

func (q *GroupedQuery) AddOp(o Op) {
	q.op = o
}

func (q *GroupedQuery) Op() Op {
	return q.op
}

func (q *GroupedQuery) AddField(s Field) {
	if config.Config.ConvertSearch {
		s = Field(util.PgSearchQueryKey(string(s)))
	}
	q.field = s
}

func (q *GroupedQuery) AddTerm(s Term) {
	tlen := len(q.terms)
	if (tlen == 0) || (q.terms[tlen-1].term != "") {
		t := QueryTerm{mod: OpNotAnOp, term: s}
		q.terms = append(q.terms, t)
	} else {
		q.terms[tlen-1].term = s
	}
}

func (q *GroupedQuery) AddTermOp(o Op) {
	t := QueryTerm{mod: o, term: ""}
	q.terms = append(q.terms, t)
}

func (q *GroupedQuery) SetNext(n Queryable) {
	q.next = n
}

func (q *GroupedQuery) Next() Queryable {
	return q.next
}

func (q *GroupedQuery) SetPrev(n Queryable) {
	q.prev = n
}

func (q *GroupedQuery) Prev() Queryable {
	return q.prev
}

func (q *GroupedQuery) IsIncomplete() bool {
	return !q.complete
}

func (q *GroupedQuery) SetCompleted() {
	q.complete = true
}

func (q *GroupedQuery) AddFuzzBoost(o Op) {
	q.terms[len(q.terms)-1].fuzzboost = o
}

func (q *GroupedQuery) AddFuzzParam(s string) {
	q.terms[len(q.terms)-1].fuzzparam = s
}

func (q *RangeQuery) AddOp(o Op) {
	q.op = o
}

func (q *RangeQuery) Op() Op {
	return q.op
}

func (q *RangeQuery) AddField(s Field) {
	if config.Config.ConvertSearch {
		s = Field(util.PgSearchQueryKey(string(s)))
	}
	q.field = s
}

func (q *RangeQuery) AddTerm(s Term) {
	if q.start == "" {
		q.start = RangeTerm(s)
	} else {
		q.end = RangeTerm(s)
	}
	q.SetCompleted()
}

func (q *RangeQuery) AddTermOp(o Op) {
	// nop
}

func (q *RangeQuery) SetNext(n Queryable) {
	q.next = n
}

func (q *RangeQuery) Next() Queryable {
	return q.next
}

func (q *RangeQuery) SetPrev(n Queryable) {
	q.prev = n
}

func (q *RangeQuery) Prev() Queryable {
	return q.prev
}

func (q *RangeQuery) IsIncomplete() bool {
	if q.start == "" || q.end == "" {
		return true
	}
	return false
}

func (q *RangeQuery) SetCompleted() {
	q.complete = true
}

func (q *RangeQuery) AddFuzzBoost(o Op) {
	// no-op
}

func (q *RangeQuery) AddFuzzParam(s string) {

}

func (q *GroupedQuery) SearchIndex(org *organization.Organization, idxName string) (map[string]indexer.Document, error) {
	tmpRes := make([]groupQueryHolder, len(q.terms))
	for i, v := range q.terms {
		tmpRes[i].op = v.mod
		notop := false
		if v.mod == OpUnaryNot || v.mod == OpUnaryPro {
			notop = true
		}
		searchTerm := makeSearchTerm(q.field, v.term)
		ix := indexer.GetIndex()
		r, err := ix.Search(org, idxName, searchTerm, notop)
		if err != nil {
			return nil, err
		}
		tmpRes[i].res = r
	}
	res, err := mergeResults(tmpRes)
	return res, err
}

func mergeResults(tmpRes []groupQueryHolder) (map[string]indexer.Document, error) {
	reqOp := false
	res := make(map[string]indexer.Document)
	var req map[string]indexer.Document

	// Merge the results, taking into account any + operators lurking about
	for _, t := range tmpRes {
		if t.op == OpUnaryReq {
			reqOp = true
			if req == nil {
				req = t.res
			} else {
				for k := range req {
					if _, found := t.res[k]; !found {
						delete(req, k)
					}
				}
			}
		} else if !reqOp {
			for k, v := range t.res {
				res[k] = v
			}
		}
	}
	if reqOp {
		req = res
	}
	return res, nil
}

func (q *RangeQuery) SearchIndex(org *organization.Organization, idxName string) (map[string]indexer.Document, error) {
	i := indexer.GetIndex()
	res, err := i.SearchRange(org, idxName, string(q.field), string(q.start), string(q.end), q.inclusive, q.negated)
	return res, err
}

func (q *SubQuery) SearchIndex(org *organization.Organization, idxName string) (map[string]indexer.Document, error) {
	return nil, nil
}

func (q *GroupedQuery) SearchResults(curRes map[string]indexer.Document) (map[string]indexer.Document, error) {
	tmpRes := make([]groupQueryHolder, len(q.terms))
	for i, v := range q.terms {
		tmpRes[i].op = v.mod
		notop := false
		if v.mod == OpUnaryNot || v.mod == OpUnaryPro {
			notop = true
		}
		searchTerm := makeSearchTerm(q.field, v.term)
		ix := indexer.GetIndex()
		r, err := ix.SearchResults(searchTerm, notop, curRes)
		if err != nil {
			return nil, err
		}
		tmpRes[i].res = r
	}
	res, err := mergeResults(tmpRes)
	return res, err
}

func (q *RangeQuery) SearchResults(curRes map[string]indexer.Document) (map[string]indexer.Document, error) {
	i := indexer.GetIndex()
	res, err := i.SearchResultsRange(string(q.field), string(q.start), string(q.end), q.inclusive, q.negated, curRes)
	return res, err
}

func (q *SubQuery) SearchResults(curRes map[string]indexer.Document) (map[string]indexer.Document, error) {
	return nil, nil
}

func (q *SubQuery) AddOp(o Op) {
	q.op = o
}

func (q *SubQuery) Op() Op {
	return q.op
}

func (q *SubQuery) AddField(s Field) {

}

func (q *SubQuery) AddTerm(s Term) {

}

func (q *SubQuery) AddTermOp(o Op) {

}

func (q *SubQuery) SetNext(n Queryable) {
	q.next = n
}

func (q *SubQuery) Next() Queryable {
	return q.next
}

func (q *SubQuery) SetPrev(n Queryable) {
	q.prev = n
}

func (q *SubQuery) Prev() Queryable {
	return q.prev
}

func (q *SubQuery) IsIncomplete() bool {
	return !q.complete
}

func (q *SubQuery) SetCompleted() {
	q.complete = true
}
func (q *SubQuery) AddFuzzBoost(o Op) {

}

func (q *SubQuery) AddFuzzParam(s string) {

}

func (q *NotQuery) SearchIndex(org *organization.Organization, idxName string) (map[string]indexer.Document, error) {
	if q.Next() == nil {
		err := errors.New("No next link present in query chain after unary NOT operator!")
		return nil, err
	}
	return q.Next().SearchIndex(org, idxName)
}

func (q *NotQuery) SearchResults(results map[string]indexer.Document) (map[string]indexer.Document, error) {
	if q.Next() == nil {
		err := errors.New("No next link present in query chain searching results after unary NOT operator!")
		return nil, err
	}
	return q.Next().SearchResults(results)
}

func (q *NotQuery) AddOp(op Op) {
	q.op = op
}

func (q *NotQuery) Op() Op {
	return q.op
}

func (q *NotQuery) AddField(f Field) {
	// noop
	return
}

func (q *NotQuery) AddTerm(t Term) {
	// noop
	return
}

func (q *NotQuery) AddTermOp(op Op) {
	// noop
	return
}

func (q *NotQuery) SetNext(n Queryable) {
	q.next = n
}

func (q *NotQuery) Next() Queryable {
	return q.next
}

func (q *NotQuery) SetPrev(n Queryable) {
	q.prev = n
}

func (q *NotQuery) Prev() Queryable {
	return q.prev
}

func (q *NotQuery) IsIncomplete() bool {
	return false
}

func (q *NotQuery) SetCompleted() {
	// noop
	return
}

func (q *NotQuery) AddFuzzBoost(op Op) {
	// noop
	return
}

func (q *NotQuery) AddFuzzParam(s string) {
	// noop
	return
}

func (z *Token) AddOp(o Op) {
	z.Latest.AddOp(o)
}

func (z *Token) AddField(s string) {
	z.Latest.AddField(Field(s))
}

func (z *Token) AddTerm(s string) {
	if z.Latest == nil || (z.Latest != nil && !z.Latest.IsIncomplete()) {
		z.StartBasic()
	}
	z.Latest.AddTerm(Term(s))
}

func (z *Token) AddTermOp(o Op) {
	if z.Latest == nil || (z.Latest != nil && !z.Latest.IsIncomplete()) {
		z.StartBasic()
	}
	z.Latest.AddTermOp(o)
}

func (z *Token) AddRange(s string) {
	z.Latest.AddTerm(Term(s))
}

func (z *Token) StartBasic() {
	/* See if we need to make a new query; sometimes we don't */
	if z.Latest == nil || (z.Latest != nil && !z.Latest.IsIncomplete()) {
		un := new(BasicQuery)
		un.op = OpBinOr
		if z.Latest != nil {
			z.Latest.SetNext(un)
			un.SetPrev(z.Latest)
		}
		if z.QueryChain == nil {
			z.QueryChain = un
		}
		z.Latest = un
	}
}

func (z *Token) StartRange(inclusive bool) {
	rn := new(RangeQuery)
	rn.op = OpBinOr
	rn.inclusive = inclusive
	if z.QueryChain == nil {
		z.QueryChain = rn
	}
	if z.Latest != nil {
		z.Latest.SetNext(rn)
		rn.SetPrev(z.Latest)
		// Don't think the prohibited operator would be allowed here
		if z.Latest.Op() == OpUnaryNot {
			rn.negated = true
		}
		if z.Latest.Prev() != nil {
			rn.op = z.Latest.Prev().Op()
		} else {
			rn.op = OpNotAnOp
		}
	}
	z.Latest = rn
}

func (z *Token) StartGrouped() {
	if z.Latest == nil || (z.Latest != nil && !z.Latest.IsIncomplete()) {
		gn := new(GroupedQuery)
		gn.op = OpBinOr
		gn.terms = make([]QueryTerm, 0)
		if z.QueryChain == nil {
			z.QueryChain = gn
		}
		if z.Latest != nil {
			z.Latest.SetNext(gn)
			gn.SetPrev(z.Latest)
		}
		z.Latest = gn
	}
}

func (z *Token) SetCompleted() {
	z.Latest.SetCompleted()
}

func (z *Token) StartSubQuery() {
	// we don't want to start a subquery if we're in a field group query
	if z.Latest == nil || (z.Latest != nil && !z.Latest.IsIncomplete()) {
		sq := new(SubQuery)
		sq.start = true
		sq.complete = true
		if z.Latest != nil {
			z.Latest.SetNext(sq)
			sq.SetPrev(z.Latest)
		}
		z.Latest = sq
		if z.QueryChain == nil {
			z.QueryChain = sq
		}
	}
}

func (z *Token) EndSubQuery() {
	// we don't want to end a subquery if we're in a field group query
	if z.Latest == nil || (z.Latest != nil && !z.Latest.IsIncomplete()) {
		sq := new(SubQuery)
		sq.end = true
		sq.complete = true
		if z.Latest != nil {
			z.Latest.SetNext(sq)
			sq.SetPrev(z.Latest)
		}

		z.Latest = sq
	}
}

func (z *Token) SetNotQuery(op Op) {
	// See if we can add the negated query token without being concerned
	// about if existing queries in the query chain are complete or not
	nq := new(NotQuery)
	nq.op = op
	if z.Latest != nil {
		z.Latest.SetNext(nq)
		nq.SetPrev(z.Latest)
	}
	if z.QueryChain == nil {
		z.QueryChain = nq
	}
	z.Latest = nq
}

func (z *Token) Evaluate() Queryable {
	return z.QueryChain
}

func makeSearchTerm(field Field, term Term) string {
	return strings.Join([]string{string(field), string(term)}, ":")
}
