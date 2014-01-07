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

/* 
Data store functionality. For now, we're just keeping in memory, but optional
use of a persistent storage backend eventually is on the list of things to do
down the road. Using go-cache (https://github.com/pmylund/go-cache) for
storing our data - might be worth using for normal caching later as well.
*/
package data_store

import (
	"github.com/pmylund/go-cache"
	"github.com/ctdk/goiardi/config"
	"strings"
	"sort"
	"encoding/gob"
	"fmt"
	"bytes"
	"sync"
	"os"
	"log"
	"io/ioutil"
	"reflect"
	"compress/zlib"
)

// Main data store
type DataStore struct {
	dsc *cache.Cache
	obj_list map[string]map[string]bool
	m sync.RWMutex
}

type dsFileStore struct {
	Cache []byte
	Obj_list []byte
}

var dataStoreCache = initDataStore()

func initDataStore() *DataStore {
	ds := new(DataStore)
	ds.dsc = cache.New(0, 0)
	ds.obj_list = make(map[string]map[string]bool)
	return ds
}

func New() *DataStore {
	return dataStoreCache
}

func (ds *DataStore) make_key(key_type string, key string) string {
	var new_key []string
	new_key = append(new_key, key_type)
	new_key = append(new_key, key)
	return strings.Join(new_key, ":")
}

func (ds *DataStore) Set(key_type string, key string, val interface{}){
	ds_key := ds.make_key(key_type, key)
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.dsc.Set(ds_key, val, -1)
	ds.addToList(key_type, key)
}

func (ds *DataStore) Get(key_type string, key string) (interface {}, bool){
	ds_key := ds.make_key(key_type, key)
	ds.m.RLock()
	ds.m.RUnlock()
	val, found := ds.dsc.Get(ds_key)
	if val != nil {
		chkNilArray(val)
	}
	return val, found
}

func (ds *DataStore) Delete(key_type string, key string){
	ds_key := ds.make_key(key_type, key)
	ds.m.Lock()
	defer ds.m.Unlock()
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

// Return a list of all objects of the given type.
func (ds *DataStore) GetList(key_type string) []string{
	j := make([]string, len(ds.obj_list[key_type]))
	i := 0
	ds.m.RLock()
	defer ds.m.RUnlock()
	for k, _ := range ds.obj_list[key_type] {
		j[i] = k
		i++
	}
	sort.Strings(j)
	return j
}

// Freeze and save the data store to disk.
func (ds *DataStore) Save() error {
	// If freeze data isn't set, don't try save the data
	if !config.Config.FreezeData {
		return nil
	}
	if config.Config.DataStoreFile == "" {
		err := fmt.Errorf("Yikes! Cannot save data store to disk because no file was specified.")
		return err
	}
	fp, err := ioutil.TempFile("", "ds-store")
	if err != nil {
		return err
	}
	zfp := zlib.NewWriter(fp)

	fstore := new(dsFileStore)
	dscache := new(bytes.Buffer)
	obj_list := new(bytes.Buffer)
	ds.m.RLock()
	defer ds.m.RUnlock()

	err = ds.dsc.Save(dscache)
	if err != nil {
		fp.Close()
		return err
	}
	enc := gob.NewEncoder(obj_list)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Something went wrong encoding the data store with Gob")
		}
	}()
	err = enc.Encode(ds.obj_list)
	if err != nil {
		fp.Close()
		return err
	}
	fstore.Cache = dscache.Bytes()
	fstore.Obj_list = obj_list.Bytes()
	enc = gob.NewEncoder(zfp)
	err = enc.Encode(fstore)
	zfp.Close()
	if err != nil {
		fp.Close()
		return err
	}
	err = fp.Close()
	if err != nil {
		return err
	}
	return os.Rename(fp.Name(), config.Config.DataStoreFile)
}

// Load the frozen data store from disk.
func (ds *DataStore) Load() error {
	// If freeze data isn't set, don't try loading the data
	if !config.Config.FreezeData {
		return nil
	}
	if config.Config.DataStoreFile == "" {
		err := fmt.Errorf("Yikes! Cannot load data store from disk because no file was specified.")
		return err
	}

	fp, err := os.Open(config.Config.DataStoreFile)
	if err != nil {
		// It's fine for the file not to exist on startup
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	zfp, zerr := zlib.NewReader(fp)
	if zerr != nil {
		fp.Close()
		return zerr
	}
	dec := gob.NewDecoder(zfp)
	ds.m.Lock()
	defer ds.m.Unlock()
	fstore := new(dsFileStore)
	err = dec.Decode(&fstore)
	zfp.Close()
	if err != nil {
		fp.Close()
		log.Printf("error at fstore")
		return err
	}

	dscache := bytes.NewBuffer(fstore.Cache)
	obj_list := bytes.NewBuffer(fstore.Obj_list)

	err = ds.dsc.Load(dscache)
	if err != nil {
		log.Println("error at dscache")
		fp.Close()
		return err
	}
	dec = gob.NewDecoder(obj_list)
	err = dec.Decode(&ds.obj_list)
	if err != nil {
		log.Println("error at obj_list")
		fp.Close()
		return err
	}
	return fp.Close()
}

func chkNilArray(obj interface{}) {
	s := reflect.ValueOf(obj).Elem()
	for i := 0; i < s.NumField(); i++ {
		v := s.Field(i)
		switch v.Kind() {
			case reflect.Slice:
				if v.IsNil(){
					o := reflect.MakeSlice(v.Type(), 0, 0)
					v.Set(o)
				}
			case reflect.Map:
				m := v.Interface()
				m = WalkMapForNil(m)
				g := reflect.ValueOf(m)
				v.Set(g)
		}
	}
}

// Walk through the given map, searching for nil slices to create. This does
// not handle all possible cases, but it *does* handle the cases found with the
// chef objects in goiardi.
func WalkMapForNil(r interface{}) interface{} {
	switch m := r.(type) {
		case map[string]interface{}:
			for k, v := range m {
				m[k] = WalkMapForNil(v)
			}
			r = m
			return r
		case []string:
			if m == nil {
				m = make([]string, 0)
			} 
			r = m
			return r
		case []interface{}:
			if m == nil {
				m = make([]interface{}, 0)
			}
			r = m
			return r
		default:
			return r
	}
}
