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

// Package indexer indexes objects that implement the Indexable interface. The
// index is all in memory right now, but it can be frozen and saved to disk for
// persistence.
package indexer

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"github.com/ctdk/go-trie/gtrie"
	"github.com/ctdk/goas/v2/logger"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Indexable is an interface that provides all the information necessary to
// index an object. All objects that will be indexed need to implement this.
type Indexable interface {
	DocID() string
	Index() string
	Flatten() []string
	OrgName() string
}

// Index holds a map of document collections.
type Index struct {
	m      sync.RWMutex
	idxmap map[string]map[string]*IdxCollection
}

// IdxCollection holds a map of documents.
type IdxCollection struct {
	m    sync.RWMutex
	docs map[string]*IdxDoc
}

// IdxDoc is the indexed documents that are actually searched.
type IdxDoc struct {
	m       sync.RWMutex
	trie    *gtrie.Node
	docText string
}

/* Index methods */

// CreateOrgDex makes an organization's index.
func CreateOrgDex(orgName string) error {
	indexMap.m.Lock()
	indexMap.checkOrCreateOrgDex(orgName)
	indexMap.m.Unlock()
	indexMap.makeDefaultCollections(orgName)
	return nil
}

// DeleteOrgDex deletes an organization's index.
func DeleteOrgDex(orgName string) error {
	indexMap.m.Lock()
	defer indexMap.m.Unlock()
	if _, ok := indexMap.idxmap[orgName]; !ok {
		return fmt.Errorf("Organization index %s not found", orgName)
	}
	delete(indexMap.idxmap, orgName)
	return nil
}

// CreateNewCollection creates an index for data bags when they are created,
// rather than when the first data bag item is uploaded
func CreateNewCollection(orgName string, idxName string) {
	indexMap.m.Lock()
	defer indexMap.m.Unlock()
	indexMap.createCollection(orgName, idxName)
}

// DeleteCollection deletes a collection from the index. Useful only for data
// bags.
func DeleteCollection(orgName string, idxName string) error {
	/* Don't try and delete built-in indexes */
	if idxName == "node" || idxName == "client" || idxName == "environment" || idxName == "role" {
		err := fmt.Errorf("%s is a default search index, cannot be deleted.", idxName)
		return err
	}
	indexMap.deleteCollection(orgName, idxName)
	return nil
}

// DeleteItemFromCollection deletes an item from a collection
func DeleteItemFromCollection(orgName string, idxName string, doc string) error {
	err := indexMap.deleteItem(orgName, idxName, doc)
	return err
}

func (i *Index) checkOrCreateOrgDex(orgName string) {
	if _, ok := i.idxmap[orgName]; !ok {
		i.idxmap[orgName] = make(map[string]*IdxCollection)
	}
}

func (i *Index) createCollection(orgName, idxName string) {
	i.checkOrCreateOrgDex(orgName)
	if _, ok := i.idxmap[orgName][idxName]; !ok {
		i.idxmap[orgName][idxName] = new(IdxCollection)
		i.idxmap[orgName][idxName].docs = make(map[string]*IdxDoc)
	}
}

func (i *Index) deleteCollection(orgName, idxName string) {
	i.m.Lock()
	defer i.m.Unlock()
	i.checkOrCreateOrgDex(orgName)
	delete(i.idxmap[orgName], idxName)
}

func (i *Index) saveIndex(object Indexable) {
	/* Have to check to see if data bag indexes exist */
	i.m.Lock()
	defer i.m.Unlock()
	i.checkOrCreateOrgDex(object.OrgName())
	if _, found := i.idxmap[object.OrgName()][object.Index()]; !found {
		i.createCollection(object.OrgName(), object.Index())
	}
	i.idxmap[object.OrgName()][object.Index()].addDoc(object)
}

func (i *Index) deleteItem(orgName, idxName string, doc string) error {
	i.m.Lock()
	defer i.m.Unlock()
	i.checkOrCreateOrgDex(orgName)
	if _, found := i.idxmap[orgName][idxName]; !found {
		err := fmt.Errorf("Index collection %s not found", idxName)
		return err
	}
	i.idxmap[orgName][idxName].delDoc(doc)
	return nil
}

func (i *Index) search(orgName string, idx string, term string, notop bool) (map[string]*IdxDoc, error) {
	i.m.Lock()
	i.checkOrCreateOrgDex(orgName)
	i.m.Unlock()

	i.m.RLock()
	defer i.m.RUnlock()

	idc, found := i.idxmap[orgName][idx]
	if !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	}
	// Special case - if term is '*:*', just return all of the keys
	if term == "*:*" {
		return idc.docs, nil
	}
	results, err := idc.searchCollection(term, notop)
	return results, err
}

func (i *Index) searchText(orgName string, idx string, term string, notop bool) (map[string]*IdxDoc, error) {
	i.m.Lock()
	i.checkOrCreateOrgDex(orgName)
	i.m.Unlock()

	i.m.RLock()
	defer i.m.RUnlock()

	idc, found := i.idxmap[orgName][idx]
	if !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	}
	results, err := idc.searchTextCollection(term, notop)
	return results, err
}

func (i *Index) searchRange(orgName string, idx string, field string, start string, end string, inclusive bool) (map[string]*IdxDoc, error) {
	i.m.Lock()
	i.checkOrCreateOrgDex(orgName)
	i.m.Unlock()

	i.m.RLock()
	defer i.m.RUnlock()

	idc, found := i.idxmap[orgName][idx]
	if !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	}
	results, err := idc.searchRange(field, start, end, inclusive)
	return results, err
}

func (i *Index) endpoints(orgName string) []string {
	i.m.Lock()
	i.checkOrCreateOrgDex(orgName)
	i.m.Unlock()

	i.m.RLock()
	defer i.m.RUnlock()

	endpoints := make([]string, len(i.idxmap[orgName]))
	n := 0
	for k := range i.idxmap[orgName] {
		endpoints[n] = k
		n++
	}

	sort.Strings(endpoints)
	return endpoints
}

/* IdxCollection methods */

func (ic *IdxCollection) addDoc(object Indexable) {
	ic.m.Lock()
	if _, found := ic.docs[object.DocID()]; !found {
		ic.docs[object.DocID()] = new(IdxDoc)
	}
	ic.m.Unlock()
	ic.m.RLock()
	defer ic.m.RUnlock()
	ic.docs[object.DocID()].update(object)
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
	rsafe := safeSearchResults(results)
	return rsafe, nil
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
	rsafe := safeSearchResults(results)
	return rsafe, nil
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
	rsafe := safeSearchResults(results)
	return rsafe, nil
}

func safeSearchResults(results map[string]*IdxDoc) map[string]*IdxDoc {
	rsafe := make(map[string]*IdxDoc, len(results))
	for k, v := range results {
		j := &v
		rsafe[k] = *j
	}
	return rsafe
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
		if e := recover(); e != nil {
			logger.Errorf("There was a problem creating the trie: %s", fmt.Sprintln(e))
		}
	}()
	trie, err := gtrie.Create(flattened)
	if err != nil {
		logger.Errorf(err.Error())
	} else {
		idoc.trie = trie
		idoc.docText = flatText
	}
}

// Examine searches a document, determining if it needs to do a search for an
// exact term or a regexp search.
func (idoc *IdxDoc) Examine(term string) (bool, error) {
	idoc.m.RLock()
	defer idoc.m.RUnlock()

	r := regexp.MustCompile(`\*|\?`)
	if r.MatchString(term) {
		m, err := idoc.regexSearch(term)
		return m, err
	}
	m := idoc.exactSearch(term)
	return m, nil
}

// TextSearch performs a text search of an index document.
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

// RangeSearch searches for a range of values.
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
	im.makeDefaultCollections("default")

	return im
}

func (i *Index) makeDefaultCollections(orgName string) {
	defaults := [...]string{"client", "environment", "node", "role"}
	i.m.Lock()
	defer i.m.Unlock()
	i.idxmap = make(map[string]map[string]*IdxCollection)
	for _, d := range defaults {
		i.createCollection(orgName, d)
	}
}

// IndexObj processes and adds an object to the index.
func IndexObj(object Indexable) {
	go indexMap.saveIndex(object)
}

// SearchIndex searches for a string in the given index. Returns a slice of
// names of matching objects, or an error on failure.
func SearchIndex(orgName, idxName string, term string, notop bool) (map[string]*IdxDoc, error) {
	res, err := indexMap.search(orgName, idxName, term, notop)
	return res, err
}

// SearchText performs a full-ish text search of the index.
func SearchText(orgName, idxName string, term string, notop bool) (map[string]*IdxDoc, error) {
	res, err := indexMap.searchText(orgName, idxName, term, notop)
	return res, err
}

// SearchRange performs a range search on the given index.
func SearchRange(orgName, idxName string, field string, start string, end string, inclusive bool) (map[string]*IdxDoc, error) {
	res, err := indexMap.searchRange(orgName, idxName, field, start, end, inclusive)
	return res, err
}

// Endpoints returns a list of currently indexed endpoints.
func Endpoints(orgName string) []string {
	endpoints := indexMap.endpoints(orgName)
	return endpoints
}

// SaveIndex saves the index files to disk.
func SaveIndex(idxFile string) error {
	return indexMap.save(idxFile)
}

// LoadIndex loads index files from disk.
func LoadIndex(idxFile string) error {
	return indexMap.load(idxFile)
}

/* gob encoding functions for the index */

func (i *Index) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	i.m.RLock()
	defer i.m.RUnlock()
	err := encoder.Encode(i.idxmap)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (i *Index) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	return decoder.Decode(&i.idxmap)
}

func (ic *IdxCollection) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	ic.m.RLock()
	defer ic.m.RUnlock()
	err := encoder.Encode(ic.docs)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (ic *IdxCollection) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	return decoder.Decode(&ic.docs)
}

func (idoc *IdxDoc) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	idoc.m.RLock()
	defer idoc.m.RUnlock()
	err := encoder.Encode(idoc.trie)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(idoc.docText)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (idoc *IdxDoc) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err := decoder.Decode(&idoc.trie)
	if err != nil {
		return err
	}
	return decoder.Decode(&idoc.docText)
}

func (i *Index) save(idxFile string) error {
	if idxFile == "" {
		err := fmt.Errorf("Yikes! Cannot save index to disk because no file was specified.")
		return err
	}
	fp, err := ioutil.TempFile(path.Dir(idxFile), "idx-build")
	if err != nil {
		return err
	}
	zfp := zlib.NewWriter(fp)
	i.m.RLock()
	defer i.m.RUnlock()
	enc := gob.NewEncoder(zfp)
	err = enc.Encode(i)
	zfp.Close()
	if err != nil {
		fp.Close()
		return err
	}
	err = fp.Close()
	if err != nil {
		return err
	}
	return os.Rename(fp.Name(), idxFile)
}

func (i *Index) load(idxFile string) error {
	if idxFile == "" {
		err := fmt.Errorf("Yikes! Cannot load index from disk because no file was specified.")
		return err
	}
	fp, err := os.Open(idxFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	zfp, zerr := zlib.NewReader(fp)
	if zerr != nil {
		fp.Close()
		return zerr
	}
	dec := gob.NewDecoder(zfp)
	err = dec.Decode(&i)
	zfp.Close()
	if err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

// ClearIndex of all collections and documents
func ClearIndex() {
	iOld := indexMap
	iOld.m.Lock()
	defer iOld.m.Unlock()
	i := new(Index)
	i.makeDefaultCollections("default")
	indexMap = i

	return
}

// ReIndex rebuilds the search index from scratch
func ReIndex(objects []Indexable) error {
	for _, o := range objects {
		indexMap.saveIndex(o)
	}
	// We really ought to be able to return from an error, but at the moment
	// there aren't any ways it does so in the index save bits.
	return nil
}
