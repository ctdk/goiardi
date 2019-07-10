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

// Package indexer indexes objects that implement the Indexable interface. The
// index is all in memory right now, but it can be frozen and saved to disk for
// persistence.
package indexer

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/ctdk/goiardi/config"
	"github.com/tideland/golib/logger"
)

var riM *sync.Mutex

func init() {
	riM = new(sync.Mutex)
}

// Indexable is an interface that provides all the information necessary to
// index an object. All objects that will be indexed need to implement this.
type Indexable interface {
	DocID() string
	Index() string
	OrgName() string
	Flatten() map[string]interface{}
}

type IndexerOrg interface {
	GetName() string
	GetId() int64
	SearchSchemaName() string
}

// Index holds a map of document collections.
type Index interface {
	Search(string, string, string, bool) (map[string]Document, error)
	SearchText(string, string, string, bool) (map[string]Document, error)
	SearchRange(string, string, string, string, string, bool, bool) (map[string]Document, error)
	SearchResults(string, bool, map[string]Document) (map[string]Document, error)
	SearchResultsRange(string, string, string, bool, bool, map[string]Document) (map[string]Document, error)
	SearchResultsText(string, bool, map[string]Document) (map[string]Document, error)
	Save() error
	Load() error
	ObjIndexer
}

type ObjIndexer interface {
	Initialize(IndexerOrg) error
	CreateOrgDex(IndexerOrg) error
	DeleteOrgDex(IndexerOrg) error
	CreateCollection(IndexerOrg, string) error
	CreateNewCollection(IndexerOrg, string) error
	DeleteCollection(IndexerOrg, string) error
	DeleteItem(IndexerOrg, string, string) error
	SaveItem(IndexerOrg, Indexable) error
	Endpoints(IndexerOrg) ([]string, error)
	OrgList() []string
	Clear(IndexerOrg) error
}

type Document interface {
}

var indexMap Index
var objIndex ObjIndexer

func Initialize(config *config.Conf, defaultOrg IndexerOrg) {
	if config.PgSearch {
		objIndex = new(PostgresIndex)
		objIndex.Initialize(defaultOrg)
	} else {
		fileindex := new(FileIndex)
		fileindex.file = config.IndexFile
		im := Index(fileindex)
		im.Initialize(defaultOrg)
		indexMap = im
		objIndex = im
	}
}

func GetIndex() Index {
	// right now just return the index map
	return indexMap
}

// CreateOrgDex makes an organization's index.
func CreateOrgDex(org IndexerOrg) error {
	return objIndex.CreateOrgDex(org)
}

// DeleteOrgDex deletes an organization's index.
func DeleteOrgDex(org IndexerOrg) error {
	return objIndex.CreateOrgDex(org)
}

// CreateNewCollection creates an index for data bags when they are created,
// rather than when the first data bag item is uploaded
func CreateNewCollection(org IndexerOrg, idxName string) {
	objIndex.CreateNewCollection(org, idxName)
}

// DeleteCollection deletes a collection from the index. Useful only for data
// bags.
func DeleteCollection(org IndexerOrg, idxName string) error {
	/* Don't try and delete built-in indexes */
	if idxName == "node" || idxName == "client" || idxName == "environment" || idxName == "role" {
		err := fmt.Errorf("%s is a default search index, cannot be deleted.", idxName)
		return err
	}
	return objIndex.DeleteCollection(org, idxName)
}

// DeleteItemFromCollection deletes an item from a collection
func DeleteItemFromCollection(org IndexerOrg, idxName string, doc string) error {
	err := objIndex.DeleteItem(org, idxName, doc)
	return err
}

// IndexObj processes and adds an object to the index.
func IndexObj(org IndexerOrg, object Indexable) {
	go objIndex.SaveItem(org, object)
}

// Endpoints returns a list of currently indexed endpoints.
func Endpoints(org IndexerOrg) ([]string, error) {
	endpoints, err := objIndex.Endpoints(org)
	return endpoints, err
}

func OrgList() []string {
	return objIndex.OrgList()
}

// SaveIndex saves the index files to disk.
func SaveIndex() error {
	// TODO: do better
	if config.Config.PgSearch {
		return nil
	}
	return indexMap.Save()
}

// LoadIndex loads index files from disk.
func LoadIndex() error {
	if config.Config.PgSearch {
		return nil
	}
	return indexMap.Load()
}

// ClearIndex of all collections and documents
func ClearIndex(org IndexerOrg) {
	err := objIndex.Clear(org)
	if err != nil {
		logger.Errorf("Error clearing db for reindexing: %s", err.Error())
	}
	return
}

// ReIndex rebuilds the search index from scratch
func ReIndex(objects []Indexable, rCh chan struct{}) error {
	go func() {
		z := 0
		t := "(none)"
		if len(objects) > 0 {
			z = len(objects)
			t = fmt.Sprintf("%T", objects[0])
			logger.Debugf("starting to reindex %d objects of %s type", z, t)
		} else {
			logger.Debugf("No objects actually in this round of reindexing")
		}
		// take the mutex
		logger.Debugf("attempting to take indexer.ReIndex mutex (%d %s)", z, t)
		riM.Lock()
		logger.Debugf("indexer.ReIndex mutex (%d %s) taken", z, t)
		mCh := make(chan struct{}, 1)
		defer func() {
			<-mCh
			logger.Debugf("releasing indexer.ReIndex mutex (%d %s)", z, t)
			rCh <- struct{}{}
			riM.Unlock()
		}()
		ch := make(chan Indexable, runtime.NumCPU())
		fCh := make(chan struct{}, z)
		for i := 0; i < runtime.NumCPU(); i++ {
			go func() {
				for obj := range ch {
					objIndex.SaveItem(obj)
					fCh <- struct{}{}
				}
				return
			}()
		}

		for _, o := range objects {
			ch <- o
		}
		close(ch)

		if z > 0 {
			for y := 0; y < z; y++ {
				<-fCh
			}
		}
		mCh <- struct{}{}
	}()
	// We really ought to be able to return from an error, but at the moment
	// there aren't any ways it does so in the index save bits.
	return nil
}
