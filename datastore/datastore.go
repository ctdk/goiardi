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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/pmylund/go-cache"
	"github.com/tideland/golib/logger"
)

// ErrorNodeStatus is for errors specific to the absence of node statuses in the
// system
type ErrorNodeStatus error

// Errors that may come up with the node statuses.
var (
	// ErrNoStatuses is returned where there are no node statuses in the
	// datastore at all.
	ErrNoStatuses ErrorNodeStatus = errors.New("No statuses in the datastore")

	// ErrNoStatusList is returned when there are statuses in the datastore,
	// but somehow the map of int slices associating a status with a node is
	// missing.
	ErrNoStatusList ErrorNodeStatus = errors.New("No status lists in the datastore")
)

// DataStore is the main data store struct, holding the key/value store and list
// of objects.
type DataStore struct {
	dsc     *cache.Cache
	objList map[string]map[string]bool
	m       sync.RWMutex
	updated bool
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
	ds.updated = true

	valBytes, err := encodeSafeVal(val)
	if err != nil {
		log.Fatalln(err)
	}
	ds.dsc.Set(dsKey, valBytes, -1)

	ds.addToList(keyType, key)
}

// Get a value of the given type associated with the given key, if it exists.
func (ds *DataStore) Get(keyType string, key string) (interface{}, bool) {
	var val interface{}
	var found bool

	dsKey := ds.makeKey(keyType, key)
	ds.m.RLock()
	defer ds.m.RUnlock()

	valEnc, f := ds.dsc.Get(dsKey)
	found = f

	if valEnc != nil {
		var err error
		val, err = decodeSafeVal(valEnc)
		if err != nil {
			log.Fatalln(err)
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
	ds.updated = true
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

func (ds *DataStore) GetListLen(keyType string) int {
	return len(ds.objList[keyType])
}

// SetNodeStatus updates a node's status using the in-memory data store.
func (ds *DataStore) SetNodeStatus(nodeName string, orgName string, obj interface{}, nsID ...int) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.updated = true
	nsKey := ds.makeKey(joinStr("nodestatus-", orgName), "nodestatuses")
	nsListKey := ds.makeKey(joinStr("nodestatuslist-", orgName), "nodestatuslists")
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

	n, err := encodeSafeVal(obj)
	if err != nil {
		return err
	}
	ns[nextID] = n

	nslist[nodeName] = append(nslist[nodeName], nextID)

	ds.dsc.Set(nsKey, ns, -1)
	ds.dsc.Set(nsListKey, nslist, -1)
	return nil
}

// ReplaceNodeStatuses replaces the node statuses being stored in the data store
// with the provided statuses that have been ordered by age already. This is
// most useful when purging old statuses.
func (ds *DataStore) ReplaceNodeStatuses(nodeName string, orgName string, objs []interface{}) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.updated = true

	// Delete the old statuses
	err := ds.deleteStatuses(nodeName, orgName)
	if err != nil {
		return err
	}

	// and put the ones we want to keep, if any, back in.
	if len(objs) == 0 {
		return nil
	}
	nsKey := ds.makeKey(joinStr("nodestatus-", orgName), "nodestatuses")
	nsListKey := ds.makeKey(joinStr("nodestatuslist-", orgName), "nodestatuslists")

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

	for _, o := range objs {
		nextID := getNextID(ns)
		n, err := encodeSafeVal(o)
		if err != nil {
			return err
		}
		ns[nextID] = n
		nslist[nodeName] = append(nslist[nodeName], nextID)
	}
	ds.dsc.Set(nsKey, ns, -1)
	ds.dsc.Set(nsListKey, nslist, -1)
	return nil
}

// AllNodeStatuses returns a list of all statuses known for the given node from
// the in-memory data store.
func (ds *DataStore) AllNodeStatuses(nodeName string, orgName string) ([]interface{}, error) {
	ds.m.RLock()
	defer ds.m.RUnlock()
	nsKey := ds.makeKey(joinStr("nodestatus-", orgName), "nodestatuses")
	nsListKey := ds.makeKey(joinStr("nodestatuslist-", orgName), "nodestatuslists")
	a, _ := ds.dsc.Get(nsKey)
	if a == nil {
		return nil, ErrNoStatuses
	}
	ns := a.(map[int]interface{})
	a, _ = ds.dsc.Get(nsListKey)
	if a == nil {
		return nil, ErrNoStatusList
	}
	nslist := a.(map[string][]int)
	arr := make([]interface{}, len(nslist[nodeName]))
	for i, v := range nslist[nodeName] {
		n, err := decodeSafeVal(ns[v])
		if err != nil {
			return nil, err
		}
		arr[i] = n
	}
	return arr, nil
}

// LatestNodeStatus returns the latest status for a node from the in-memory
// data store.
func (ds *DataStore) LatestNodeStatus(nodeName string, orgName string) (interface{}, error) {
	ds.m.RLock()
	defer ds.m.RUnlock()
	nsKey := ds.makeKey(joinStr("nodestatus-", orgName), "nodestatuses")
	nsListKey := ds.makeKey(joinStr("nodestatuslist-", orgName), "nodestatuslists")
	a, _ := ds.dsc.Get(nsKey)
	if a == nil {
		return nil, ErrNoStatuses
	}
	ns := a.(map[int]interface{})
	a, _ = ds.dsc.Get(nsListKey)
	if a == nil {
		return nil, ErrNoStatusList
	}
	nslist := a.(map[string][]int)
	nsarr := nslist[nodeName]
	if nsarr == nil {
		err := fmt.Errorf("no statuses found for node %s", nodeName)
		return nil, ErrorNodeStatus(err)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(nsarr)))
	var n interface{}
	var err error

	n, err = decodeSafeVal(ns[nsarr[0]])
	if err != nil {
		return nil, err
	}

	return n, nil
}

// DeleteNodeStatus deletes all status reports for a node from the in-memory
// data store.
func (ds *DataStore) DeleteNodeStatus(nodeName string, orgName string) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.updated = true
	return ds.deleteStatuses(nodeName, orgName)
}

func (ds *DataStore) deleteStatuses(nodeName string, orgName string) error {
	nsKey := ds.makeKey(joinStr("nodestatus-", orgName), "nodestatuses")
	nsListKey := ds.makeKey(joinStr("nodestatuslist-", orgName), "nodestatuslists")
	a, _ := ds.dsc.Get(nsKey)
	if a == nil {
		return ErrNoStatuses
	}
	ns := a.(map[int]interface{})
	a, _ = ds.dsc.Get(nsListKey)
	if a == nil {
		return ErrNoStatusList
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

func (ds *DataStore) getLogInfoMap(orgName string) map[int]interface{} {
	dsKey := ds.makeKey(joinStr("loginfo-", orgName), "loginfos")
	var a interface{}
	aEnc, _ := ds.dsc.Get(dsKey)
	if aEnc != nil {
		var err error
		a, err = decodeSafeVal(aEnc)
		if err != nil {
			log.Fatalln(err)
		}
	}
	if a == nil {
		a = make(map[int]interface{})
	}
	arr := a.(map[int]interface{})
	return arr
}

func (ds *DataStore) setLogInfoMap(orgName string, liMap map[int]interface{}) {
	dsKey := ds.makeKey(joinStr("loginfo-", orgName), "loginfos")
	valBytes, err := encodeSafeVal(liMap)
	if err != nil {
		log.Fatalln(err)
	}
	ds.dsc.Set(dsKey, valBytes, -1)
}

// SetLogInfo sets a loginfo in the data store. Unlike most of these objects,
// log infos are stored and retrieved by id, since they have no useful names.
func (ds *DataStore) SetLogInfo(orgName string, obj interface{}, logID ...int) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.updated = true
	arr := ds.getLogInfoMap(orgName)
	var nextID int
	if logID != nil {
		nextID = logID[0]
	} else {
		nextID = getNextID(arr)
	}
	arr[nextID] = obj
	ds.setLogInfoMap(orgName, arr)
	return nil
}

// DeleteLogInfo deletes a logged event from the data store.
func (ds *DataStore) DeleteLogInfo(orgName string, id int) error {
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.updated = true
	arr := ds.getLogInfoMap(orgName)
	delete(arr, id)
	ds.setLogInfoMap(orgName, arr)
	return nil
}

// PurgeLogInfoBefore purges all the logged events with an id less than the one
// given from the data store.
func (ds *DataStore) PurgeLogInfoBefore(orgName string, id int) (int64, error) {
	ds.m.Lock()
	defer ds.m.Unlock()
	ds.updated = true
	arr := ds.getLogInfoMap(orgName)
	newLogs := make(map[int]interface{})
	var purged int64
	for k, v := range arr {
		if k > id {
			newLogs[k] = v
		} else {
			purged++
		}
	}
	ds.setLogInfoMap(orgName, newLogs)
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
func (ds *DataStore) GetLogInfo(orgName string, id int) (interface{}, error) {
	ds.m.RLock()
	defer ds.m.RUnlock()
	arr := ds.getLogInfoMap(orgName)
	item := arr[id]
	if item == nil {
		err := fmt.Errorf("Log info with id %d not found", id)
		return nil, err
	}
	return item, nil
}

// GetLogInfoList gets all the log infos currently stored.
func (ds *DataStore) GetLogInfoList(orgName string) map[int]interface{} {
	ds.m.RLock()
	defer ds.m.RUnlock()
	arr := ds.getLogInfoMap(orgName)
	return arr
}

// Save freezes and saves the data store to disk.
func (ds *DataStore) Save(dsFile string) error {
	if !ds.updated {
		return nil
	}
	logger.Debugf("Data has changed, saving data store to disk")
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
	ds.updated = false

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

// TODO: Is the below even needed anymore?

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

func joinStr(str ...string) string {
	return strings.Join(str, "")
}

func (ds *DataStore) SetAssociationReq(name string, variant string, key string, obj interface{}) {
	ds.m.Lock()
	defer ds.m.Unlock()
	a := ds.getAssocReqMap(name, variant)
	a[key] = obj
	ds.setAssocReqMap(name, variant, a)
	return
}

func (ds *DataStore) GetAssociationReqs(name string, variant string) []interface{} {
	ds.m.Lock()
	defer ds.m.Unlock()
	a := ds.getAssocReqMap(name, variant)
	var l []interface{}
	if len(a) > 0 {
		l = make([]interface{}, len(a))
		n := 0
		for _, v := range a {
			l[n] = v
			n++
		}
	}
	return l
}

func (ds *DataStore) DelAssociationReq(name string, variant string, key string) {
	ds.m.Lock()
	defer ds.m.Unlock()
	a := ds.getAssocReqMap(name, variant)
	delete(a, key)
	ds.setAssocReqMap(name, variant, a)
	return
}

func (ds *DataStore) DelAllAssociationReqs(name string, variant string) {
	ds.m.Lock()
	defer ds.m.Unlock()
	dsKey := ds.makeKey(joinStr("assocreqmap-", variant), name)
	ds.dsc.Delete(dsKey)
}

func (ds *DataStore) getAssocReqMap(name, variant string) map[string]interface{} {
	return ds.getAssocMapBase("assocreqmap", name, variant)
}

func (ds *DataStore) setAssocReqMap(name, variant string, associations map[string]interface{}) {
	ds.setAssocMapBase("assocreqmap", name, variant, associations)
}

func (ds *DataStore) getAssocMap(name, variant string) map[string]interface{} {
	return ds.getAssocMapBase("assocmap", name, variant)
}

func (ds *DataStore) setAssocMap(name, variant string, associations map[string]interface{}) {
	ds.setAssocMapBase("assocmap", name, variant, associations)
}

func (ds *DataStore) getAssocMapBase(cont string, name string, variant string) map[string]interface{} {
	dsKey := ds.makeKey(joinStr(cont, "-", variant), name)
	var a interface{}
	aEnc, _ := ds.dsc.Get(dsKey)
	if aEnc != nil {
		var err error
		a, err = decodeSafeVal(aEnc)
		if err != nil {
			log.Fatalln(err)
		}
	}
	if a == nil {
		a = make(map[string]interface{})
	}
	assoc := a.(map[string]interface{})
	return assoc

}

func (ds *DataStore) setAssocMapBase(cont string, name string, variant string, associations map[string]interface{}) {
	dsKey := ds.makeKey(joinStr(cont, "-", variant), name)
	valBytes, err := encodeSafeVal(associations)
	if err != nil {
		log.Fatalln(err)
	}
	ds.dsc.Set(dsKey, valBytes, -1)
}

func (ds *DataStore) SetAssociation(name string, variant string, key string, obj interface{}) {
	ds.m.Lock()
	defer ds.m.Unlock()
	a := ds.getAssocMap(name, variant)
	a[key] = obj
	ds.setAssocMap(name, variant, a)
	return
}

func (ds *DataStore) GetAssociations(name string, variant string) []interface{} {
	ds.m.Lock()
	defer ds.m.Unlock()
	a := ds.getAssocMap(name, variant)
	var l []interface{}
	if len(a) > 0 {
		l = make([]interface{}, len(a))
		n := 0
		for _, v := range a {
			l[n] = v
			n++
		}
	}
	return l
}

func (ds *DataStore) DelAssociation(name string, variant string, key string) {
	ds.m.Lock()
	defer ds.m.Unlock()
	a := ds.getAssocMap(name, variant)
	delete(a, key)
	ds.setAssocMap(name, variant, a)
	return
}

func (ds *DataStore) DelAllAssociations(name string, variant string) {
	ds.m.Lock()
	defer ds.m.Unlock()
	dsKey := ds.makeKey(joinStr("assocmap-", variant), name)
	ds.dsc.Delete(dsKey)
}
