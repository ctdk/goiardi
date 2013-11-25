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

/* 
Data store functionality. For now, we're just keeping in memory, but optional
use of a persistent storage backend eventually is on the list of things to do
down the road. Using go-cache (https://github.com/pmylund/go-cache) for
storing our data - might be worth using for normal caching later as well.
*/
package data_store

import (
	"github.com/pmylund/go-cache"
	"strings"
	"sort"
)

type DataStore struct {
	dsc *cache.Cache
	obj_list map[string]map[string]bool
}

var data_store_cache *cache.Cache
var object_list map[string]map[string]bool

func New() *DataStore {
	ds := new(DataStore)
	if data_store_cache == nil {
		/* We want stuff in the data store until we explicitly remove
		 * it. */
		data_store_cache = cache.New(0, 0)
	}
	ds.dsc = data_store_cache
	if object_list == nil {
		object_list = make(map[string]map[string]bool)
	}
	ds.obj_list = object_list
	return ds
}

func (ds *DataStore) make_key(key_type string, key string) string {
	var new_key []string
	new_key = append(new_key, key_type)
	new_key = append(new_key, key)
	return strings.Join(new_key, ":")
}

func (ds *DataStore) Set(key_type string, key string, val interface{}){
	ds_key := ds.make_key(key_type, key)
	ds.dsc.Set(ds_key, val, -1)
	ds.addToList(key_type, key)
}

func (ds *DataStore) Get(key_type string, key string) (interface {}, bool){
	ds_key := ds.make_key(key_type, key)
	val, found := ds.dsc.Get(ds_key)
	return val, found
}

func (ds *DataStore) Delete(key_type string, key string){
	ds_key := ds.make_key(key_type, key)
	ds.dsc.Delete(ds_key)
	ds.removeFromList(key_type, key)
}

/* For the in-memory data store stuff, we need a convenient list of objects,
 * since it's not a database and we can't just pull that up. This won't be
 * useful normally. */

func (ds *DataStore) addToList(key_type string, key string){
	if ds.obj_list[key_type] == nil {
		ds.obj_list[key_type] = make(map[string]bool)
	}
	ds.obj_list[key_type][key] = true
}

func (ds *DataStore) removeFromList(key_type string, key string){
	if ds.obj_list[key_type] != nil {
		/* If it's nil, we don't have to worry about deleting the key */
		delete(ds.obj_list[key_type], key)
	}
}

func (ds *DataStore) GetList(key_type string) []string{
	j := make([]string, len(ds.obj_list[key_type]))
	i := 0
	for k, _ := range ds.obj_list[key_type] {
		j[i] = k
		i++
	}
	sort.Strings(j)
	return j
}
