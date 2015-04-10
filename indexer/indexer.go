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
	"fmt"
	"github.com/ctdk/goiardi/config"
)

// Indexable is an interface that provides all the information necessary to
// index an object. All objects that will be indexed need to implement this.
type Indexable interface {
	DocID() string
	Index() string
	Flatten() map[string]interface{}
}

// Index holds a map of document collections.
type Index interface {
	Initialize()
	Search(string, string, bool) (map[string]*Document, error)
	SearchText(string, string, bool) (map[string]*Document, error)
	SearchRange(string, string, string, string, bool) (map[string]*Document, error)
	SearchResults(string, bool, map[string]*Document) (map[string]*Document, error)
	SearchResultsRange(string, string, string, bool, map[string]*Document) (map[string]*Document, error)
	SearchResultsText(string, bool, map[string]*Document) (map[string]*Document, error)
	Save() error
	Load() error
	ObjIndexer
}

type ObjIndexer interface {
	CreateCollection(string)
	DeleteCollection(string) error
	DeleteItem(string, string) error
	SaveItem(Indexable)
	Endpoints() []string
	Clear()

}

type Document interface {
}

var indexMap Index
var objIndex ObjIndexer

func Initialize(config *config.Conf) {
	fileindex := new(FileIndex)
	fileindex.file = config.IndexFile

	im := Index(fileindex)
	im.Initialize()

	indexMap = im
	objIndex = im
}

func GetIndex() Index {
	// right now just return the index map
	return indexMap
}

// CreateNewCollection creates an index for data bags when they are created,
// rather than when the first data bag item is uploaded
func CreateNewCollection(idxName string) {
	objIndex.CreateCollection(idxName)
}

// DeleteCollection deletes a collection from the index. Useful only for data
// bags.
func DeleteCollection(idxName string) error {
	/* Don't try and delete built-in indexes */
	if idxName == "node" || idxName == "client" || idxName == "environment" || idxName == "role" {
		err := fmt.Errorf("%s is a default search index, cannot be deleted.", idxName)
		return err
	}
	return objIndex.DeleteCollection(idxName)
}

// DeleteItemFromCollection deletes an item from a collection
func DeleteItemFromCollection(idxName string, doc string) error {
	err := objIndex.DeleteItem(idxName, doc)
	return err
}


// IndexObj processes and adds an object to the index.
func IndexObj(object Indexable) {
	go objIndex.SaveItem(object)
}

// Endpoints returns a list of currently indexed endpoints.
func Endpoints() []string {
	endpoints := objIndex.Endpoints()
	return endpoints
}

// SaveIndex saves the index files to disk.
func SaveIndex() error {
	return indexMap.Save()
}

// LoadIndex loads index files from disk.
func LoadIndex() error {
	return indexMap.Load()
}

// ClearIndex of all collections and documents
func ClearIndex() {
	objIndex.Clear()
	return
}

// ReIndex rebuilds the search index from scratch
func ReIndex(objects []Indexable) error {
	for _, o := range objects {
		objIndex.SaveItem(o)
	}
	// We really ought to be able to return from an error, but at the moment
	// there aren't any ways it does so in the index save bits.
	return nil
}
