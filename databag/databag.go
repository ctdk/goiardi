/* Data bags! */

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

// Package databag provides a convenient way to store arbitrary data on the
// server.
package databag

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
)

// DataBag is the overall data bag.
type DataBag struct {
	Name         string
	DataBagItems map[string]*DataBagItem
	id           int32
}

// DataBagItem is an individual item within a data bag.
type DataBagItem struct {
	Name        string                 `json:"name"`
	ChefType    string                 `json:"chef_type"`
	JSONClass   string                 `json:"json_class"`
	DataBagName string                 `json:"data_bag"`
	RawData     map[string]interface{} `json:"raw_data"`
	id          int32
	dataBagID   int32
	origName    string
}

/* Data bag functions and methods */

// New creates an empty data bag, and kicks off adding it to the index.
func New(name string) (*DataBag, util.Gerror) {
	var found bool
	var err util.Gerror

	if err = validateDataBagName(name, false); err != nil {
		return nil, err
	}

	if config.UsingDB() {
		var cerr error
		found, cerr = checkForDataBagSQL(datastore.Dbh, name)
		if cerr != nil {
			err = util.Errorf(cerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := datastore.New()
		_, found = ds.Get("data_bag", name)
	}
	if found {
		err = util.Errorf("Data bag %s already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	dbiMap := make(map[string]*DataBagItem)
	dataBag := &DataBag{
		Name:         name,
		DataBagItems: dbiMap,
	}
	indexer.CreateNewCollection(name)
	return dataBag, nil
}

// Get a data bag.
func Get(dbName string) (*DataBag, util.Gerror) {
	var dataBag *DataBag
	var err error
	if config.UsingDB() {
		dataBag, err = getDataBagSQL(dbName)
		if err != nil {
			var gerr util.Gerror
			if err == sql.ErrNoRows {
				gerr = util.Errorf("Cannot load data bag %s", dbName)
				gerr.SetStatus(http.StatusNotFound)
			} else {
				gerr = util.Errorf(err.Error())
				gerr.SetStatus(http.StatusInternalServerError)
			}
			return nil, gerr
		}
	} else {
		ds := datastore.New()
		d, found := ds.Get("data_bag", dbName)
		if !found {
			err := util.Errorf("Cannot load data bag %s", dbName)
			err.SetStatus(http.StatusNotFound)
			return nil, err
		}
		if d != nil {
			dataBag = d.(*DataBag)
			for _, v := range dataBag.DataBagItems {
				z := datastore.WalkMapForNil(v.RawData)
				v.RawData = z.(map[string]interface{})
			}
		}
	}
	return dataBag, nil
}

// Save a data bag.
func (db *DataBag) Save() error {
	if config.Config.UseMySQL {
		return db.saveMySQL()
	} else if config.Config.UsePostgreSQL {
		return db.savePostgreSQL()
	} else {
		ds := datastore.New()
		ds.Set("data_bag", db.Name, db)
	}
	return nil
}

// Delete a data bag.
func (db *DataBag) Delete() error {
	if config.UsingDB() {
		err := db.deleteSQL()
		if err != nil {
			return err
		}
	} else {
		ds := datastore.New()
		/* be thorough, and remove DBItems too */
		for dbiName := range db.DataBagItems {
			db.DeleteDBItem(dbiName)
		}
		ds.Delete("data_bag", db.Name)
	}
	indexer.DeleteCollection(db.Name)
	return nil
}

// GetList returns a list of data bags on the server.
func GetList() []string {
	var dbList []string
	if config.UsingDB() {
		dbList = getListSQL()
	} else {
		ds := datastore.New()
		dbList = ds.GetList("data_bag")
	}
	return dbList
}

// GetName returns the data bag's name.
func (db *DataBag) GetName() string {
	return db.Name
}

// URLType returns the base element of a data bag's URL.
func (db *DataBag) URLType() string {
	return "data"
}

// GetName returns the data bag item's identifier.
func (dbi *DataBagItem) GetName() string {
	return dbi.DocID()
}

// URLType returns the base element of a data bag's URL.
func (dbi *DataBagItem) URLType() string {
	return "data"
}

/* Data bag item functions and methods */

// NewDBItem creates a new data bag item in the associated data bag.
func (db *DataBag) NewDBItem(rawDbagItem map[string]interface{}) (*DataBagItem, util.Gerror) {
	var dbiID string
	var dbagItem *DataBagItem
	switch t := rawDbagItem["id"].(type) {
	case string:
		if t == "" {
			err := util.Errorf("Field 'id' missing")
			return nil, err
		}
		dbiID = t
	default:
		err := util.Errorf("Field 'id' missing")
		return nil, err
	}
	if err := validateDataBagName(dbiID, true); err != nil {
		return nil, err
	}
	dbiFullName := fmt.Sprintf("data_bag_item_%s_%s", db.Name, dbiID)

	if config.UsingDB() {
		d, err := db.getDBItemSQL(dbiID)
		if d != nil || (err != nil && err != sql.ErrNoRows) {
			if err != nil {
				logger.Debugf("Log real SQL error in NewDBItem: %s", err.Error())
			}
			gerr := util.Errorf("Data Bag Item '%s' already exists in Data Bag '%s'.", dbiID, db.Name)
			gerr.SetStatus(http.StatusConflict)
			return nil, gerr
		}
		if config.Config.UseMySQL {
			dbagItem, err = db.newDBItemMySQL(dbiID, rawDbagItem)
		} else if config.Config.UsePostgreSQL {
			dbagItem, err = db.newDBItemPostgreSQL(dbiID, rawDbagItem)
		}
		if err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		/* Look for an existing dbag item with this name */
		if d, _ := db.GetDBItem(dbiID); d != nil {
			gerr := util.Errorf("Data Bag Item '%s' already exists in Data Bag '%s'.", dbiID, db.Name)
			gerr.SetStatus(http.StatusConflict)
			return nil, gerr
		}
		/* But should we store the raw data as a JSON string? */
		dbagItem = &DataBagItem{
			Name:        dbiFullName,
			ChefType:    "data_bag_item",
			JSONClass:   "Chef::DataBagItem",
			DataBagName: db.Name,
			RawData:     rawDbagItem,
		}
		db.DataBagItems[dbiID] = dbagItem
	}
	err := db.Save()
	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	indexer.IndexObj(dbagItem)
	return dbagItem, nil
}

// UpdateDBItem updates a data bag item in this data bag.
func (db *DataBag) UpdateDBItem(dbiID string, rawDbagItem map[string]interface{}) (*DataBagItem, error) {
	dbItem, err := db.GetDBItem(dbiID)
	if err != nil {
		if err == sql.ErrNoRows {
			err = fmt.Errorf("Cannot load data bag item %s for data bag %s", dbiID, db.Name)
		}
		return nil, err
	}
	dbItem.RawData = rawDbagItem
	if config.UsingDB() {
		err = dbItem.updateDBItemSQL()
		if err != nil {
			return nil, err
		}
	} else {
		db.DataBagItems[dbiID] = dbItem
	}
	err = db.Save()
	if err != nil {
		return nil, err
	}
	indexer.IndexObj(dbItem)
	return dbItem, nil
}

// DeleteDBItem deletes a data bag item.
func (db *DataBag) DeleteDBItem(dbItemName string) error {
	if config.UsingDB() {
		dbi, err := db.GetDBItem(dbItemName)
		if err != nil {
			return err
		}
		err = dbi.deleteDBItemSQL()
		if err != nil {
			return err
		}
	} else {
		delete(db.DataBagItems, dbItemName)
	}
	err := db.Save()
	if err != nil {
		return err
	}
	indexer.DeleteItemFromCollection(db.Name, dbItemName)
	return nil
}

// GetDBItem gets a data bag item.
func (db *DataBag) GetDBItem(dbItemName string) (*DataBagItem, error) {
	if config.UsingDB() {
		dbi, err := db.getDBItemSQL(dbItemName)
		if err == sql.ErrNoRows {
			err = fmt.Errorf("data bag item %s in %s not found", dbItemName, db.Name)
		}
		return dbi, err
	}
	dbi, ok := db.DataBagItems[dbItemName]
	if !ok {
		err := fmt.Errorf("data bag item %s in %s not found", dbItemName, db.Name)
		return nil, err
	}
	return dbi, nil
}

// GetMultiDBItems gets multiple data bag items from a slice of names.
func (db *DataBag) GetMultiDBItems(dbItemNames []string) ([]*DataBagItem, util.Gerror) {
	var dbis []*DataBagItem
	if config.UsingDB() {
		var err error
		dbis, err = db.getMultiDBItemSQL(dbItemNames)
		if err != nil && err != sql.ErrNoRows {
			return nil, util.CastErr(err)
		}
	} else {
		dbis = make([]*DataBagItem, 0, len(dbItemNames))
		for _, d := range dbItemNames {
			do, _ := db.DataBagItems[d]
			if do != nil {
				dbis = append(dbis, do)
			}
		}
	}
	return dbis, nil
}

// AllDBItems returns a map of all the items in a data bag.
func (db *DataBag) AllDBItems() (map[string]*DataBagItem, error) {
	if config.UsingDB() {
		return db.allDBItemsSQL()
	}
	return db.DataBagItems, nil
}

// ListDBItems returns a list of items in a data bag.
func (db *DataBag) ListDBItems() []string {
	if config.UsingDB() {
		return db.listDBItemsSQL()
	}
	dbis := make([]string, len(db.DataBagItems))
	n := 0
	for k := range db.DataBagItems {
		dbis[n] = k
		n++
	}
	return dbis
}

// NumDBItems returns the number of items in a data bag.
func (db *DataBag) NumDBItems() int {
	if config.UsingDB() {
		return db.numDBItemsSQL()
	}
	return len(db.DataBagItems)
}

func (db *DataBag) fullDBItemName(dbItemName string) string {
	return fmt.Sprintf("data_bag_item_%s_%s", db.Name, dbItemName)
}

// RawDataBagJSON extract the data bag item's raw data from the request, saving
// it to the server.
func RawDataBagJSON(data io.ReadCloser) map[string]interface{} {
	rawDbagItem := make(map[string]interface{})
	dec := json.NewDecoder(data)
	dec.UseNumber()

	dec.Decode(&rawDbagItem)
	var rawData map[string]interface{}

	/* The way data can come from knife may
	 * not be entirely consistent. Use
	 * raw data from the json hash if we
	 * have it, otherwise assume it's just
	 * the raw data without the other chef
	 * stuff added. */

	if _, ok := rawDbagItem["raw_data"]; ok {
		rawData = rawDbagItem["raw_data"].(map[string]interface{})
	} else {
		rawData = rawDbagItem
	}
	return rawData
}

func validateDataBagName(name string, dbi bool) util.Gerror {
	item := "name"
	if dbi {
		item = "id"
	}
	_ = item // may want this later
	if !util.ValidateDBagName(name) {
		err := util.Errorf("Field '%s' invalid", item)
		err.SetStatus(http.StatusBadRequest)
		return err
	}
	return nil
}

/* Indexing functions for data bag items */

// DocID returns the id of the data bag item for the indexer.
func (dbi *DataBagItem) DocID() string {
	switch did := dbi.RawData["id"].(type) {
	case string:
		return did
	default:
		d := strings.Replace(dbi.Name, dbi.DataBagName, "", 1)
		return d
	}
}

// Index returns the name of the data bag this data bag item belongs to, so it's
// placed in the correct index.
func (dbi *DataBagItem) Index() string {
	return dbi.DataBagName
}

// Flatten a data bag item out so it's suitable for indexing.
func (dbi *DataBagItem) Flatten() map[string]interface{} {
	flatten := make(map[string]interface{})
	for key, v := range dbi.RawData {
		subExpand := util.DeepMerge(key, v)
		for k, u := range subExpand {
			flatten[k] = u
		}
	}
	return flatten
}

// AllDataBags returns all data bags on this server, and all their items.
func AllDataBags() []*DataBag {
	var dataBags []*DataBag
	if config.UsingDB() {
		dataBags = allDataBagsSQL()
	} else {
		dbagList := GetList()
		for _, d := range dbagList {
			db, err := Get(d)
			if err != nil {
				continue
			}
			dataBags = append(dataBags, db)
		}
	}
	return dataBags
}
