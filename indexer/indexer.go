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
	Flatten() []string
}

// Index holds a map of document collections.
type Index interface {
	initialize()
	createCollection(string)
	deleteCollection(string)
	saveIndex(Indexable)
	deleteItem(string, string) error
	search(string, string, bool) (map[string]*Document, error)
	searchText(string, string, bool) (map[string]*Document, error)
	searchRange(string, string, string, string, bool) (map[string]*Document, error)
	endpoints() []string
	makeDefaultCollections()
	save() error
	load() error
}

type Document interface {
}

type IndexCollection interface {
	addDoc(Indexable)
	delDoc(string)
	allDocs() map[string]*Document
	searchCollection(string, bool) (map[string]*Document, error)
	searchTextCollection(string, bool) (map[string]*Document, error)
	searchRange(string, string, string, bool) (map[string]*Document, error)
}

var indexMap Index

func Initialize(config *config.Conf) {
	fileindex := new(FileIndex)
	fileindex.file = config.IndexFile

	im := Index(fileindex)
	im.makeDefaultCollections()
	im.initialize()

	indexMap = im
}

// Create a new index collection.

// CreateNewCollection creates an index for data bags when they are created,
// rather than when the first data bag item is uploaded
func CreateNewCollection(idxName string) {
	indexMap.createCollection(idxName)
}

// DeleteCollection deletes a collection from the index. Useful only for data
// bags.
func DeleteCollection(idxName string) error {
	/* Don't try and delete built-in indexes */
	if idxName == "node" || idxName == "client" || idxName == "environment" || idxName == "role" {
		err := fmt.Errorf("%s is a default search index, cannot be deleted.", idxName)
		return err
	}
	indexMap.deleteCollection(idxName)
	return nil
}

// DeleteItemFromCollection deletes an item from a collection
func DeleteItemFromCollection(idxName string, doc string) error {
	err := indexMap.deleteItem(idxName, doc)
	return err
}

// IndexObj processes and adds an object to the index.
func IndexObj(object Indexable) {
	go indexMap.saveIndex(object)
}

// SearchIndex searches for a string in the given index. Returns a slice of
// names of matching objects, or an error on failure.
func SearchIndex(idxName string, term string, notop bool) (map[string]*Document, error) {
	res, err := indexMap.search(idxName, term, notop)
	return res, err
}

// SearchText performs a full-ish text search of the index.
func SearchText(idxName string, term string, notop bool) (map[string]*Document, error) {
	res, err := indexMap.searchText(idxName, term, notop)
	return res, err
}

// SearchRange performs a range search on the given index.
func SearchRange(idxName string, field string, start string, end string, inclusive bool) (map[string]*Document, error) {
	res, err := indexMap.searchRange(idxName, field, start, end, inclusive)
	return res, err
}

// Endpoints returns a list of currently indexed endpoints.
func Endpoints() []string {
	endpoints := indexMap.endpoints()
	return endpoints
}

// SaveIndex saves the index files to disk.
func SaveIndex() error {
	return indexMap.save()
}

// LoadIndex loads index files from disk.
func LoadIndex() error {
	return indexMap.load()
}

// ClearIndex of all collections and documents
func ClearIndex() {
	indexMap.makeDefaultCollections()
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
