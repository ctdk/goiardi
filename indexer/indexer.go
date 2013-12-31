/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
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

//Index objects that implement the Indexable interface. All in memory right now,
//but the possibility is open to later saving the index to disk for later use if
//a full solr installation (with the attendant improved functionality, 
//admittedly) is not needed.
package indexer

import (
	"github.com/ctdk/go-trie/gtrie"
	"log"
	"sync"
	"strings"
	"sort"
	"fmt"
	"regexp"
)

type Indexable interface {
	DocId() string
	Index() string
	Flatten() []string
}

type Index struct {
	m sync.RWMutex
	idxmap map[string]*IdxCollection
}

type IdxCollection struct {
	m sync.RWMutex
	docs map[string]*IdxDoc
}

type IdxDoc struct {
	m sync.RWMutex
	trie *gtrie.Node
	docText string
}

/* Index methods */

// Create a new index collection.

// Create an index for data bags when they are created, rather than when the
// first data bag item is uploaded
func CreateNewCollection(idxName string) {
	indexMap.createCollection(idxName)
}

// Delete a collection from the index. Useful only for data bags.
func DeleteCollection(idxName string) error {
	/* Don't try and delete built-in indexes */
	if idxName == "node" || idxName == "client" || idxName == "environment" || idxName == "client" {
		err := fmt.Errorf("%s is a default search index, cannot be deleted.", idxName)
		return err
	}
	indexMap.deleteCollection(idxName)
	return nil
}

// Delete an item from a collection
func DeleteItemFromCollection(idxName string, doc string) error {
	err := indexMap.deleteItem(idxName, doc)
	return err
}

func (i *Index) createCollection(idxName string) {
	i.m.Lock()
	defer i.m.Unlock()
	/* It's not inconceivable that a previous check for the existence of
	 * the index collection had a new index collection created under it,
	 * so only make a new one if it doesn't exist. */
	if _, ok := i.idxmap[idxName]; !ok {
		i.idxmap[idxName] = new(IdxCollection)
		i.idxmap[idxName].docs = make(map[string]*IdxDoc)
	}
}

func (i *Index) deleteCollection(idxName string) {
	i.m.Lock()
	defer i.m.Unlock()
	delete(i.idxmap, idxName)
}

func (i *Index) saveIndex(object Indexable) {
	/* Have to check to see if data bag indexes exist */
	if _, found := i.idxmap[object.Index()]; !found {
		i.createCollection(object.Index())
	}
	i.idxmap[object.Index()].addDoc(object)
}

func (i *Index) deleteItem(idxName string, doc string) error {
	if _, found := i.idxmap[idxName]; !found {
		err := fmt.Errorf("Index collection %s not found", idxName)
		return err
	}
	i.idxmap[idxName].delDoc(doc)
	return nil
}

func (i *Index) search(idx string, term string, notop bool) (map[string]*IdxDoc, error){
	if idc, found := i.idxmap[idx]; !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	} else {
		// Special case - if term is '*:*', just return all of the
		// keys
		if term == "*:*" {
			return idc.docs, nil
		} 
		results, err := idc.searchCollection(term, notop)
		return results, err
	}
}

func (i *Index) searchText(idx string, term string, notop bool) (map[string]*IdxDoc, error) {
	if idc, found := i.idxmap[idx]; !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	} else {
		results, err := idc.searchTextCollection(term, notop)
		return results, err
	}
}

func (i *Index) searchRange(idx string, field string, start string, end string, inclusive bool) (map[string]*IdxDoc, error){
	if idc, found := i.idxmap[idx]; !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	} else {
		results, err := idc.searchRange(field, start, end, inclusive)
		return results, err
	}
}

func (i *Index) endpoints() []string {
	i.m.RLock()
	defer i.m.RUnlock()

	endpoints := make([]string, len(i.idxmap))
	n := 0
	for k := range i.idxmap {
		endpoints[n] = k
		n++
	}

	sort.Strings(endpoints)
	return endpoints
}

/* IdxCollection methods */

func (ic *IdxCollection) addDoc(object Indexable) {
	ic.m.RLock()
	defer ic.m.RUnlock()
	
	if _, found := ic.docs[object.DocId()]; !found {
		ic.docs[object.DocId()] = new(IdxDoc)
	}
	ic.docs[object.DocId()].update(object)
}

func (ic *IdxCollection) delDoc(doc string) {
	ic.m.Lock()
	defer ic.m.Unlock()
	
	delete(ic.docs, doc)
}

/* Search for an exact key/value match */
func (ic *IdxCollection) searchCollection(term string, notop bool) (map[string]*IdxDoc, error) {
	results := make(map[string]*IdxDoc)
	ic.m.RLock()
	defer ic.m.RUnlock()
	for k, v := range ic.docs {
		m, err := v.Examine(term)
		if err != nil {
			return nil, err
		}
		if (m && !notop) || (!m && notop) {
			results[k] = v
		}
	}
	return results, nil
}

func (ic *IdxCollection) searchTextCollection(term string, notop bool) (map[string]*IdxDoc, error) {
	results := make(map[string]*IdxDoc)
	ic.m.RLock()
	defer ic.m.RUnlock()
	for k, v := range ic.docs {
		m, err := v.TextSearch(term)
		if err != nil {
			return nil, err
		}
		if (m && !notop) || (!m && notop) {
			results[k] = v
		}
	}
	return results, nil
}

func (ic *IdxCollection) searchRange(field string, start string, end string, inclusive bool) (map[string]*IdxDoc, error) {
	results := make(map[string]*IdxDoc)
	ic.m.RLock()
	defer ic.m.RUnlock()
	
	for k, v := range ic.docs {
		m, err := v.RangeSearch(field, start, end, inclusive)
		if err != nil {
			return nil, err
		}
		if m {
			results[k] = v
		}
	}
	return results, nil
}

/* IdxDoc methods */
func (idoc *IdxDoc) update(object Indexable) {
	idoc.m.Lock()
	defer idoc.m.Unlock()
	flattened := object.Flatten()
	flatText := strings.Join(flattened, "\n")
	/* recover from horrific trie errors that seem to happen with really
	 * big values. :-/ */
	defer func() {
		if e:= recover(); e != nil {
			log.Println("There was a problem creating the trie.")
			log.Println(e)
		}
	}()
	trie, err := gtrie.Create(flattened)
	if err != nil {
		log.Println(err)
	} else {
		idoc.trie = trie
		idoc.docText = flatText
	}
}

func (idoc *IdxDoc) Examine(term string) (bool, error) {
	idoc.m.RLock()
	defer idoc.m.RUnlock()
	
	r := regexp.MustCompile(`\*|\?`)
	if r.MatchString(term) {
		m, err := idoc.regexSearch(term)
		return m, err
	} else {
		m := idoc.exactSearch(term)
		return m, nil
	}
}

func (idoc *IdxDoc) TextSearch(term string) (bool, error) {
	if term[0] == '*' || term[0] == '?' {
		err := fmt.Errorf("Can't start a term with a wildcard character")
		return false, err
	}
	term = strings.Replace(term, "*", ".*", -1)
	term = strings.Replace(term, "?", ".?", -1)
	re := fmt.Sprintf("(?m):%s$", term)
	reComp, err := regexp.Compile(re)
	if err != nil {
		return false, err
	}
	idoc.m.RLock()
	defer idoc.m.RUnlock()
	m := reComp.MatchString(idoc.docText)
	return m, nil
}

func (idoc *IdxDoc) RangeSearch(field string, start string, end string, inclusive bool) (bool, error) {
	// The parser should catch a lot of possible errors, happily

	// "*" is permitted as a range that indicates anything bigger or smaller
	// than the other range, depending
	wildStart := false
	wildEnd := false
	if start == "*" {
		wildStart = true
	}
	if end == "*" {
		wildEnd = true
	}
	if wildStart && wildEnd {
		err := fmt.Errorf("you can't have both start and end be wild in a range search, sadly")
		return false, err
	}
	idoc.m.RLock()
	defer idoc.m.RUnlock()
	key := fmt.Sprintf("%s:", field)
	if n, _ := idoc.trie.HasPrefix(key); n != nil {
		kids := n.ChildKeys()
		for _, child := range kids {
			if inclusive {
				if wildStart {
					if child <= end {
						return true, nil
					}
				} else if wildEnd {
					if child >= start {
						return true, nil
					}
				} else {
					if child >= start && child <= end {
						return true, nil
					}
				}
			} else {
				if wildStart {
					if child < end {
						return true, nil
					}
				} else if wildEnd {
					if child > start {
						return true, nil
					}
				} else {
					if child > start && child < end {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func (idoc *IdxDoc) exactSearch(term string) bool {
	return idoc.trie.Accepts(term)
}

func (idoc *IdxDoc) regexSearch(reTerm string) (bool, error) {
	z := strings.SplitN(reTerm, ":", 2)
	key := fmt.Sprintf("%s:", z[0])
	re := z[1]
	/* Must add . before any * or ? in the regexp first. Taking the easy way
	 * out and using strings.Replace. */
	re = strings.Replace(re, "*", ".*", -1)
	re = strings.Replace(re, "?", ".?", -1)
	reComp, err := regexp.Compile(re)
	if err != nil {
		return false, err
	}
	/* What would be better would be to fetch all of the parts of the key
	 * before the regexp part starts. Hmmm. */
	if n, _ := idoc.trie.HasPrefix(key); n != nil {
		kids := n.ChildKeys()
		for _, c := range kids {
			if reComp.MatchString(c) {
				return true, nil
			}
		}
	}
	return false, nil
}

var indexMap = initializeIndex()

func initializeIndex() *Index {
	/* We always want these indices at least. */
	im := new(Index)
	im.idxmap = make(map[string]*IdxCollection)
	defaults := [...]string{ "client", "environment", "node", "role" }
	for _, d := range defaults {
		im.createCollection(d)
	}
	return im
}

//Process and add an object to the index.
func IndexObj(object Indexable) {
	go indexMap.saveIndex(object)
}

//Search for a string in the given index. Returns a slice of names of matching
//objects, or an error on failure.
func SearchIndex(idxName string, term string, notop bool) (map[string]*IdxDoc, error) {
	res, err := indexMap.search(idxName, term, notop)
	return res, err
}

func SearchText(idxName string, term string, notop bool) (map[string]*IdxDoc, error) {
	res, err := indexMap.searchText(idxName, term, notop)
	return res, err
}

func SearchRange(idxName string, field string, start string, end string, inclusive bool) (map[string]*IdxDoc, error) {
	res, err := indexMap.searchRange(idxName, field, start, end, inclusive)
	return res, err
}

// Return a list of currently indexed endpoints
func Endpoints() []string {
	endpoints := indexMap.endpoints()
	return endpoints
}
