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
Package datastore provides data store functionality. The data store is kept in
memory, but optionally the data store may be saved to a file to provide a
perisistent data store. This uses go-cache (https://github.com/pmylund/go-cache)
for storing the data.

The methods that set, get, and delete key/value pairs also take a `keyType`
argument that specifies what kind of object it is.
*/
package datastore

import (
	"bytes"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/pmylund/go-cache"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// DataStore is the main data store struct, holding the key/value store and list
// of objects.
type DataStore struct {
	dsc     *cache.Cache
	objList map[string]map[string]bool
	m       sync.RWMutex
}

type dsFileStore struct {
	Cache   []byte
	ObjList []byte
}

type dsItem struct {
	Item interface{}
}

var dataStoreCache = initDataStore()

func initDataStore() *DataStore {
	ds := new(DataStore)
	ds.dsc = cache.New(0, 0)
	ds.objList = make(map[string]map[string]bool)
	return ds
}

// New creates a new data store instance, or returns an already created one.
func New() *DataStore {
	return dataStoreCache
}

func (ds *DataStore) makeKey(keyType string, key string) string {
	var newKey []string
	newKey = append(newKey, keyType)
	newKey = append(newKey, key)
	return strings.Join(newKey, ":")
}

// Set a value of the given type with the provided key.
func (ds *DataStore) Set(keyType string, key string, val interface{}) {
	dsKey := ds.makeKey(keyType, key)
	ds.m.Lock()
	defer ds.m.Unlock()
	if config.Config.UseUnsafeMemStore {
		ds.dsc.Set(dsKey, val, -1)
	} else {
		valBytes, err := encodeSafeVal(val)
		if err != nil {
			log.Fatalln(err)
		}
		ds.dsc.Set(dsKey, valBytes, -1)
	}
	ds.addToList(keyType, key)
}

// Get a value of the given type associated with the given key, if it exists.
func (ds *DataStore) Get(keyType string, key string) (interface{}, bool) {
	var val interface{}
	var found bool

	dsKey := ds.makeKey(keyType, key)
	ds.m.RLock()
	defer ds.m.RUnlock()

	if config.Config.UseUnsafeMemStore {
		val, found = ds.dsc.Get(dsKey)
	} else {
		valEnc, f := ds.dsc.Get(dsKey)
		found = f

		if valEnc != nil {
			var err error
			val, err = decodeSafeVal(valEnc)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	if val != nil {
		ChkNilArray(val)
	}
	return val, found
}

func encodeSafeVal(val interface{}) ([]byte, error) {
	valBuf := new(bytes.Buffer)
	valItem := &dsItem{Item: val}
	enc := gob.NewEncoder(valBuf)
	err := enc.Encode(valItem)
	
	if err != nil {
		return nil, err
	}
	return valBuf.Bytes(), nil
}

func decodeSafeVal(valEnc interface{}) (interface{}, error) {
	valBuf := bytes.NewBuffer(valEnc.([]byte))
	valItem := new(dsItem)
	dec := gob.NewDecoder(valBuf)
	err := dec.Decode(&valItem)
	if err != nil {
		return nil, err
	}
	return valItem.Item, nil
}

// Delete a value from the data store.
func (ds *DataStore) Delete(keyType string, key string) {
	dsKey := ds.makeKey(keyType, key)
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.dsc.Delete(dsKey)
	ds.removeFromList(keyType, key)
}

/* For the in-memory data store stuff, we need a convenient list of objects,
 * since it's not a database and we can't just pull that up. This won't be
 * useful normally. */

func (ds *DataStore) addToList(keyType string, key string) {
	if ds.objList[keyType] == nil {
		ds.objList[keyType] = make(map[string]bool)
	}
	ds.objList[keyType][key] = true
}

func (ds *DataStore) removeFromList(keyType string, key string) {
	if ds.objList[keyType] != nil {
		/* If it's nil, we don't have to worry about deleting the key */
		delete(ds.objList[keyType], key)
	}
}

// GetList returns a list of all objects of the given type.
func (ds *DataStore) GetList(keyType string) []string {
	j := make([]string, len(ds.objList[keyType]))
	i := 0
	ds.m.RLock()
	defer ds.m.RUnlock()
	for k := range ds.objList[keyType] {
		j[i] = k
		i++
	}
	sort.Strings(j)
	return j
}

func SetNodeStatus(nodeName string, obj interface{}, nsID ...int) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	nsKey := ds.makeKey("nodestatus", "nodestatuses")
	nsListKey := ds.makeKey("nodestatuslist", "nodestatuslists")
	a, _ := ds.dsc.Get(nsKey)
	if a == nil {
		a = make(map[int]interface{})
	}
	ns := a.(map[int]interface{})
	a, _ = ds.dsc.Get(nsListKey)
	if a == nil {
		a = make(map[string][]int)
	}
	nslist := a.(map[string][]int)
	var nextID int
	if nsID != nil {
		nextID = nsID[0]
	} else {
		nextID = getNextID(ns)
	}
	ns[nextID] = obj
	nsList[nodeName] = append(nslist[nodeName], nextID)

	ds.dsc.Set(nsKey, ns, -1)
	ds.dsc.Set(nsListKey, nslist, -1)
	return nil
}

func AllNodeStatuses(nodeName string) ([]interface{}, error) {
	ds.m.RLock()
	defer ds.m.RUnlock()
	nsKey := ds.makeKey("nodestatus", "nodestatuses")
	nsListKey := ds.makeKey("nodestatuslist", "nodestatuslists")
	a, _ := ds.dsc.Get(nsKey)
	if a == nil {
		err := fmt.Errorf("No statuses in the datastore")
		return nil, err
	}
	ns := a.(map[int]interface{})
	a, _ = ds.dsc.Get(nsListKey)
	if a == nil {
		err := fmt.Errorf("No status lists in the datastore")
		return nil, err
	}
	nslist := a.(map[string][]int)
	arr := make([]interface{}, len(nslist[nodeName]))
	for i, v := range nslist[nodeName] {
		arr[i] = v
	}
	return arr, nil
}

func LatestNodeStatus(nodeName string) (interface{}, error) {
	ds.m.RLock()
	defer ds.m.RUnlock()
	nsKey := ds.makeKey("nodestatus", "nodestatuses")
	nsListKey := ds.makeKey("nodestatuslist", "nodestatuslists")
	a, _ := ds.dsc.Get(nsKey)
	if a == nil {
		err := fmt.Errorf("No statuses in the datastore")
		return nil, err
	}
	ns := a.(map[int]interface{})
	a, _ = ds.dsc.Get(nsListKey)
	if a == nil {
		err := fmt.Errorf("No status lists in the datastore")
		return nil, err
	}
	nslist := a.(map[string][]int)
	if nslist[nodeName] == nil {
		err := fmt.Errorf("no statuses found for node %s", nodeName)
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.IntSlice(nlist[nodeName])
	return ns[nlist[nodeName][0]], nil
}

func DeleteNodeStatus(nodeName string) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	nsKey := ds.makeKey("nodestatus", "nodestatuses")
	nsListKey := ds.makeKey("nodestatuslist", "nodestatuslists")
	a, _ := ds.dsc.Get(nsKey)
	if a == nil {
		err := fmt.Errorf("No statuses in the datastore")
		return err
	}
	ns := a.(map[int]interface{})
	a, _ = ds.dsc.Get(nsListKey)
	if a == nil {
		err := fmt.Errorf("No status lists in the datastore")
		return err
	}
	nslist := a.(map[string][]int)
	for _, v := range nslist[nodeName] {
		delete(ns, v)
	}
	delete(nslist, nodeName)
	ds.dsc.Set(nsKey, ns, -1)
	ds.dsc.Set(nsListKey, nslist, -1)
	return nil
}

func (ds *DataStore) getLogInfoMap() map[int]interface{} {
	dsKey := ds.makeKey("loginfo", "loginfos")
	var a interface{}
	if config.Config.UseUnsafeMemStore {
		a, _ = ds.dsc.Get(dsKey)
	} else {
		aEnc, _ := ds.dsc.Get(dsKey)
		if aEnc != nil {
			var err error
			a, err = decodeSafeVal(aEnc)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	if a == nil {
		a = make(map[int]interface{})
	}
	arr := a.(map[int]interface{})
	return arr
}

func (ds *DataStore) setLogInfoMap(liMap map[int]interface{}) {
	dsKey := ds.makeKey("loginfo", "loginfos")
	if config.Config.UseUnsafeMemStore {
		ds.dsc.Set(dsKey, liMap, -1)
	} else {
		valBytes, err := encodeSafeVal(liMap)
		if err != nil {
			log.Fatalln(err)
		}
		ds.dsc.Set(dsKey, valBytes, -1)
	}
}

// SetLogInfo sets a loginfo in the data store. Unlike most of these objects,
// log infos are stored and retrieved by id, since they have no useful names.
func (ds *DataStore) SetLogInfo(obj interface{}, logID ...int) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	arr := ds.getLogInfoMap()
	var nextID int
	if logID != nil {
		nextID = logID[0]
	} else {
		nextID = getNextID(arr)
	}
	arr[nextID] = obj
	ds.setLogInfoMap(arr)
	return nil
}

// DeleteLogInfo deletes a logged event from the data store.
func (ds *DataStore) DeleteLogInfo(id int) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	arr := ds.getLogInfoMap()
	delete(arr, id)
	ds.setLogInfoMap(arr)
	return nil
}

// PurgeLogInfoBefore purges all the logged events with an id less than the one
// given from the data store.
func (ds *DataStore) PurgeLogInfoBefore(id int) (int64, error) {
	ds.m.Lock()
	defer ds.m.Unlock()
	arr := ds.getLogInfoMap()
	newLogs := make(map[int]interface{})
	var purged int64
	for k, v := range arr {
		if k > id {
			newLogs[k] = v
		} else {
			purged++
		}
	}
	ds.setLogInfoMap(newLogs)
	return purged, nil
}

func getNextID(lis map[int]interface{}) int {
	if len(lis) == 0 {
		return 1
	}
	var keys []int
	for k := range lis {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))
	return keys[0] + 1
}

// GetLogInfo gets a loginfo by id.
func (ds *DataStore) GetLogInfo(id int) (interface{}, error) {
	ds.m.RLock()
	defer ds.m.RUnlock()
	arr := ds.getLogInfoMap()
	item := arr[id]
	if item == nil {
		err := fmt.Errorf("Log info with id %d not found", id)
		return nil, err
	}
	return item, nil
}

// GetLogInfoList gets all the log infos currently stored.
func (ds *DataStore) GetLogInfoList() map[int]interface{} {
	ds.m.RLock()
	defer ds.m.RUnlock()
	arr := ds.getLogInfoMap()
	return arr
}

// Save freezes and saves the data store to disk.
func (ds *DataStore) Save(dsFile string) error {
	if dsFile == "" {
		err := fmt.Errorf("Yikes! Cannot save data store to disk because no file was specified.")
		return err
	}
	fp, err := ioutil.TempFile(path.Dir(dsFile), "ds-store")
	if err != nil {
		return err
	}
	zfp := zlib.NewWriter(fp)

	fstore := new(dsFileStore)
	dscache := new(bytes.Buffer)
	objList := new(bytes.Buffer)
	ds.m.RLock()
	defer ds.m.RUnlock()

	err = ds.dsc.Save(dscache)
	if err != nil {
		fp.Close()
		return err
	}
	enc := gob.NewEncoder(objList)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Something went wrong encoding the data store with Gob")
		}
	}()
	err = enc.Encode(ds.objList)
	if err != nil {
		fp.Close()
		return err
	}
	fstore.Cache = dscache.Bytes()
	fstore.ObjList = objList.Bytes()
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
	return os.Rename(fp.Name(), dsFile)
}

// Load the frozen data store from disk.
func (ds *DataStore) Load(dsFile string) error {
	if dsFile == "" {
		err := fmt.Errorf("Yikes! Cannot load data store from disk because no file was specified.")
		return err
	}

	fp, err := os.Open(dsFile)
	if err != nil {
		// It's fine for the file not to exist on startup
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
	objList := bytes.NewBuffer(fstore.ObjList)

	err = ds.dsc.Load(dscache)
	if err != nil {
		log.Println("error at dscache")
		fp.Close()
		return err
	}
	dec = gob.NewDecoder(objList)
	err = dec.Decode(&ds.objList)
	if err != nil {
		log.Println("error at objList")
		fp.Close()
		return err
	}
	return fp.Close()
}

// ChkNilArray examines an object, searching for empty slices.
// When restoring an object from either the in-memory data store after it has
// been saved to disk, or loading an object from the database with gob encoded
// data structures, empty slices are encoded as "null" when they're sent out as
// JSON to the client. This makes the client very unhappy, so those empty slices
// need to be recreated again. Annoying, but it's how it goes.
func ChkNilArray(obj interface{}) {
	s := reflect.ValueOf(obj).Elem()
	for i := 0; i < s.NumField(); i++ {
		v := s.Field(i)
		switch v.Kind() {
		case reflect.Slice:
			if v.IsNil() {
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

// WalkMapForNil walks through the given map, searching for nil slices to create.
// This does not handle all possible cases, but it *does* handle the cases found
// with the chef objects in goiardi.
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
