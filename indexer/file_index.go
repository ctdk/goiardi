package indexer

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"github.com/ctdk/go-trie/gtrie"
	"github.com/ctdk/goas/v2/logger"
	"github.com/philhofer/msgp/msgp"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"fmt"
	"sort"
	"io/ioutil"
)

type FileIndex struct {
	file    string
	m       sync.RWMutex
	idxmap  map[string]IndexCollection
	updated bool
}

// IdxCollection holds a map of documents.
type IdxCollection struct {
	m    sync.RWMutex
	docs map[string]*IdxDoc
}

// IdxDoc is the indexed documents that are actually searched.
type IdxDoc struct {
	m       sync.RWMutex
	trie    []byte
	docText []byte
}

type searchRes struct {
	key string
	doc *IdxDoc
}

/* Index methods */

func (i *FileIndex) initialize() {
	in := new(FileIndex)
	ic := new(IdxCollection)
	id := new(IdxDoc)
	gob.Register(in)
	gob.Register(ic)
	gob.Register(id)
}

func (i *FileIndex) createCollection(idxName string) {
	i.updated = true

	if _, ok := i.idxmap[idxName]; !ok {
		coll := new(IdxCollection)
		coll.docs = make(map[string]*IdxDoc)
		casted := IndexCollection(coll)
		i.idxmap[idxName] = casted
	}
}

func (i *FileIndex) deleteCollection(idxName string) {
	i.m.Lock()
	defer i.m.Unlock()
	i.updated = true
	delete(i.idxmap, idxName)
}

func (i *FileIndex) saveIndex(object Indexable) {
	/* Have to check to see if data bag indexes exist */
	i.m.Lock()
	defer i.m.Unlock()
	i.updated = true
	if _, found := i.idxmap[object.Index()]; !found {
		i.createCollection(object.Index())
	}
	i.idxmap[object.Index()].addDoc(object)
}

func (i *FileIndex) deleteItem(idxName string, doc string) error {
	i.m.Lock()
	defer i.m.Unlock()
	i.updated = true
	if _, found := i.idxmap[idxName]; !found {
		err := fmt.Errorf("Index collection %s not found", idxName)
		return err
	}
	i.idxmap[idxName].delDoc(doc)
	return nil
}

func (i *FileIndex) search(idx string, term string, notop bool) (map[string]*Document, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	idc, found := i.idxmap[idx]
	if !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	}
	// Special case - if term is '*:*', just return all of the keys
	if term == "*:*" {
		return idc.allDocs(), nil
	}
	results, err := idc.searchCollection(term, notop)
	return results, err
}

func (i *FileIndex) searchText(idx string, term string, notop bool) (map[string]*Document, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	idc, found := i.idxmap[idx]
	if !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	}
	results, err := idc.searchTextCollection(term, notop)
	return results, err
}

func (i *FileIndex) searchRange(idx string, field string, start string, end string, inclusive bool) (map[string]*Document, error) {
	i.m.RLock()
	defer i.m.RUnlock()
	idc, found := i.idxmap[idx]
	if !found {
		err := fmt.Errorf("I don't know how to search for %s data objects.", idx)
		return nil, err
	}
	results, err := idc.searchRange(field, start, end, inclusive)
	return results, err
}

func (i *FileIndex) endpoints() []string {
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

func (i *FileIndex) makeDefaultCollections() {
	defaults := [...]string{"client", "environment", "node", "role"}
	i.m.Lock()
	defer i.m.Unlock()
	i.updated = true
	i.idxmap = make(map[string]IndexCollection)
	for _, d := range defaults {
		i.createCollection(d)
	}
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
func (ic *IdxCollection) searchCollection(term string, notop bool) (map[string]*Document, error) {
	results := make(map[string]*Document)
	ic.m.RLock()
	defer ic.m.RUnlock()
	l := len(ic.docs)
	errCh := make(chan error, l)
	resCh := make(chan *searchRes, l)
	for k, v := range ic.docs {
		go func(k string, v *IdxDoc) {
			m, err := v.Examine(term)
			if err != nil {
				errCh <- err
				resCh <- nil
			} else {
				errCh <- nil
				if (m && !notop) || (!m && notop) {
					r := &searchRes{k, v}
					resCh <- r
				} else {
					resCh <- nil
				}
			}
		}(k, v)
	}
	for i := 0; i < l; i++ {
		e := <-errCh
		if e != nil {
			return nil, e
		}
	}
	for i := 0; i < l; i++ {
		r := <-resCh
		if r != nil {
			doc := Document(r.doc)
			results[r.key] = &doc
		}
	}
	rsafe := safeSearchResults(results)
	return rsafe, nil
}

func (ic *IdxCollection) searchTextCollection(term string, notop bool) (map[string]*Document, error) {
	results := make(map[string]*Document)
	ic.m.RLock()
	defer ic.m.RUnlock()
	l := len(ic.docs)
	errCh := make(chan error, l)
	resCh := make(chan *searchRes, l)
	for k, v := range ic.docs {
		go func(k string, v *IdxDoc) {
			m, err := v.TextSearch(term)
			if err != nil {
				errCh <- err
				resCh <- nil
			} else {
				errCh <- nil
				if (m && !notop) || (!m && notop) {
					r := &searchRes{k, v}
					logger.Debugf("Adding result %s to channel", k)
					resCh <- r
				} else {
					resCh <- nil
				}
			}
		}(k, v)
	}
	for i := 0; i < l; i++ {
		e := <-errCh
		if e != nil {
			return nil, e
		}
	}
	for i := 0; i < l; i++ {
		r := <-resCh
		if r != nil {
			logger.Debugf("adding result")
			doc := Document(r.doc)
			results[r.key] = &doc
		}
	}
	rsafe := safeSearchResults(results)
	return rsafe, nil
}

func (ic *IdxCollection) searchRange(field string, start string, end string, inclusive bool) (map[string]*Document, error) {
	results := make(map[string]*Document)
	ic.m.RLock()
	defer ic.m.RUnlock()
	l := len(ic.docs)
	errCh := make(chan error, l)
	resCh := make(chan *searchRes, l)
	for k, v := range ic.docs {
		go func(k string, v *IdxDoc) {
			m, err := v.RangeSearch(field, start, end, inclusive)
			if err != nil {
				errCh <- err
				resCh <- nil
			} else {
				errCh <- nil
				if m {
					r := &searchRes{k, v}
					logger.Debugf("Adding result %s to channel", k)
					resCh <- r
				} else {
					resCh <- nil
				}
			}
		}(k, v)
	}
	for i := 0; i < l; i++ {
		e := <-errCh
		if e != nil {
			return nil, e
		}
	}
	for i := 0; i < l; i++ {
		r := <-resCh
		if r != nil {
			logger.Debugf("adding result")
			doc := Document(r.doc)
			results[r.key] = &doc
		}
	}
	rsafe := safeSearchResults(results)
	return rsafe, nil
}

func safeSearchResults(results map[string]*Document) map[string]*Document {
	rsafe := make(map[string]*Document, len(results))
	for k, v := range results {
		j := &v
		rsafe[k] = *j
	}
	return rsafe
}

func (ic *IdxCollection) allDocs() map[string]*Document {
	docs := make(map[string]*Document, len(ic.docs))

	for k, v := range ic.docs {
		doc := Document(v)
		docs[k] = &doc
	}

	return docs
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
		var err error
		idoc.trie, err = compressTrie(trie)
		if err != nil {
			panic(err)
		}
		idoc.docText, err = compressText(flatText)
		if err != nil {
			panic(err)
		}
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
	m, err := idoc.exactSearch(term)
	if err != nil {
		return false, err
	}
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
	docText, err := decompressText(idoc.docText)
	if err != nil {
		return false, err
	}
	m := reComp.MatchString(docText)
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
	trie, err := decompressTrie(idoc.trie)
	if err != nil {
		return false, err
	}
	if n, _ := trie.HasPrefix(key); n != nil {
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

func (idoc *IdxDoc) exactSearch(term string) (bool, error) {
	trie, err := decompressTrie(idoc.trie)
	if err != nil {
		return false, err
	}
	return trie.Accepts(term), nil
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
	trie, err := decompressTrie(idoc.trie)
	if err != nil {
		return false, err
	}
	if n, _ := trie.HasPrefix(key); n != nil {
		kids := n.ChildKeys()
		for _, c := range kids {
			if reComp.MatchString(c) {
				return true, nil
			}
		}
	}
	return false, nil
}



/* gob encoding functions for the index */

func (i *FileIndex) GobEncode() ([]byte, error) {
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

func (i *FileIndex) GobDecode(buf []byte) error {
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

func (i *FileIndex) save() error {
	idxFile := i.file
	if idxFile == "" {
		err := fmt.Errorf("Yikes! Cannot save index to disk because no file was specified.")
		return err
	}
	if !i.updated {
		return nil
	}
	logger.Infof("Index has changed, saving to disk")
	fp, err := ioutil.TempFile(path.Dir(idxFile), "idx-build")
	if err != nil {
		return err
	}
	zfp := zlib.NewWriter(fp)
	i.m.RLock()
	defer i.m.RUnlock()
	i.updated = false
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

func (i *FileIndex) load() error {
	idxFile := i.file
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

func compressTrie(t *gtrie.Node) ([]byte, error) {
	b := new(bytes.Buffer)
	z := zlib.NewWriter(b)
	err := msgp.Encode(z, t)
	z.Close()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func decompressTrie(buf []byte) (*gtrie.Node, error) {
	b := bytes.NewBuffer(buf)
	z, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	t := new(gtrie.Node)
	err = msgp.Decode(z, t)
	err2 := z.Close()
	if err != nil {
		return nil, err
	}
	if err2 != nil {
		return nil, err2
	}
	return t, nil
}

func compressText(t string) ([]byte, error) {
	b := new(bytes.Buffer)
	z := zlib.NewWriter(b)
	_, err := z.Write([]byte(t))
	z.Close()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func decompressText(buf []byte) (string, error) {
	b := bytes.NewBuffer(buf)
	z, err := zlib.NewReader(b)
	if err != nil {
		return "", err
	}
	t, err := ioutil.ReadAll(z)
	z.Close()
	if err != nil {
		return "", err
	}
	return string(t), nil
}

