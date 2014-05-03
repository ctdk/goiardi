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

// Package data_bag provides a convenient way to store arbitrary data on the 
// server.
package data_bag

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/config"
	"fmt"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"database/sql"
	"git.tideland.biz/goas/logger"
)

// The overall data bag.
type DataBag struct {
	Name string
	DataBagItems map[string]*DataBagItem
	id int32
}

// An item within a data bag.
type DataBagItem struct {
	Name string `json:"name"`
	ChefType string `json:"chef_type"`
	JsonClass string `json:"json_class"`
	DataBagName string `json:"data_bag"`
	RawData map[string]interface{} `json:"raw_data"`
	id int32
	data_bag_id int32
	origName string
}

/* Data bag functions and methods */

func New(name string) (*DataBag, util.Gerror){
	var found bool
	var err util.Gerror

	if err = validateDataBagName(name, false); err != nil {
		return nil, err
	}

	if config.Config.UseMySQL {
		var cerr error
		found, cerr = checkForDataBagMySQL(data_store.Dbh, name)
		if cerr != nil {
			err = util.Errorf(cerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := data_store.New()
		_, found = ds.Get("data_bag", name)
	}
	if found {
		err = util.Errorf("Data bag %s already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	
	dbi_map := make(map[string]*DataBagItem)
	data_bag := &DataBag{
		Name: name,
		DataBagItems: dbi_map,
	}
	indexer.CreateNewCollection(name)
	return data_bag, nil
}

func Get(db_name string) (*DataBag, util.Gerror){
	var data_bag *DataBag
	var err error
	if config.Config.UseMySQL {
		data_bag, err = getDataBagMySQL(db_name)
		if err != nil {
			var gerr util.Gerror
			if err == sql.ErrNoRows {
				gerr = util.Errorf("Cannot load data bag %s", db_name)
				gerr.SetStatus(http.StatusNotFound)
			} else {
				gerr = util.Errorf(err.Error())
				gerr.SetStatus(http.StatusInternalServerError)
			}
			return nil, gerr
		}
	} else {
		ds := data_store.New()
		d, found := ds.Get("data_bag", db_name)
		if !found {
			err := util.Errorf("Cannot load data bag %s", db_name)
			err.SetStatus(http.StatusNotFound)
			return nil, err
		}
		if d != nil {
			data_bag = d.(*DataBag)
			for _, v := range data_bag.DataBagItems {
				z := data_store.WalkMapForNil(v.RawData)
				v.RawData = z.(map[string]interface{})
			}
		}
	}
	return data_bag, nil
}

func (db *DataBag) Save() error {
	if config.Config.UseMySQL {
		return db.saveMySQL()
	} else {
		ds := data_store.New()
		ds.Set("data_bag", db.Name, db)
	}
	return nil
}

func (db *DataBag) Delete() error {
	if config.Config.UseMySQL {
		err := db.deleteMySQL()
		if err != nil {
			return err
		}
	} else {
		ds := data_store.New()
		/* be thorough, and remove DBItems too */
		for dbiName := range db.DataBagItems {
			db.DeleteDBItem(dbiName)
		}
		ds.Delete("data_bag", db.Name)
	}
	indexer.DeleteCollection(db.Name)
	return nil
}

// Returns a list of data bags on the server.
func GetList() []string {
	var db_list []string
	if config.Config.UseMySQL {
		db_list = getListMySQL()
	} else {
		ds := data_store.New()
		db_list = ds.GetList("data_bag")
	}
	return db_list
}

func (db *DataBag) GetName() string {
	return db.Name
}

func (db *DataBag) URLType() string {
	return "data"
}

/* Data bag item functions and methods */

/* To do: Idle test; see if changes to the returned data bag item are reflected
 * in the one stored in the hash there */

// Create a new data bag item in the associated data bag.
func (db *DataBag) NewDBItem (raw_dbag_item map[string]interface{}) (*DataBagItem, util.Gerror){
	//dbi_id := raw_dbag_item["id"].(string)
	var dbi_id string
	var dbag_item *DataBagItem
	switch t := raw_dbag_item["id"].(type) {
		case string:
			if t == "" {
				err := util.Errorf("Field 'id' missing")
				return nil, err
			} else {
				dbi_id = t
			}
		default:
			err := util.Errorf("Field 'id' missing")
			return nil, err
	}
	if err := validateDataBagName(dbi_id, true); err != nil {
		return nil, err
	}
	dbi_full_name := fmt.Sprintf("data_bag_item_%s_%s", db.Name, dbi_id)

	if config.Config.UseMySQL {
		d, err := db.getDBItemMySQL(dbi_id)
		if d != nil || (err != nil && err != sql.ErrNoRows) {
			if err != nil {
				logger.Debugf("Log real SQL error in NewDBItem: %s", err.Error())
			}
			gerr := util.Errorf("Data Bag Item '%s' already exists in Data Bag '%s'.", dbi_id, db.Name)
			gerr.SetStatus(http.StatusConflict)
			return nil, gerr
		}
		dbag_item, err = db.newDBItemMySQL(dbi_id, raw_dbag_item)
		if err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
	} else {
		/* Look for an existing dbag item with this name */
		if d, _ := db.GetDBItem(dbi_id); d != nil {
			gerr := util.Errorf("Data Bag Item '%s' already exists in Data Bag '%s'.", dbi_id, db.Name)
			gerr.SetStatus(http.StatusConflict)
			return nil, gerr
		}
		/* But should we store the raw data as a JSON string? */
		dbag_item = &DataBagItem{
			Name: dbi_full_name,
			ChefType: "data_bag_item",
			JsonClass: "Chef::DataBagItem",
			DataBagName: db.Name,
			RawData: raw_dbag_item,
		}
		db.DataBagItems[dbi_id] = dbag_item
	}
	err := db.Save()
	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	indexer.IndexObj(dbag_item)
	return dbag_item, nil
}

// Updates a data bag item in this data bag.
func (db *DataBag) UpdateDBItem(dbi_id string, raw_dbag_item map[string]interface{}) (*DataBagItem, error){
	db_item, err := db.GetDBItem(dbi_id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = fmt.Errorf("Cannot load data bag item %s for data bag %s", dbi_id, db.Name)
		}
		return nil, err
	}
	db_item.RawData = raw_dbag_item
	if config.Config.UseMySQL {
		err = db_item.updateDBItemMySQL()
		if err != nil {
			return nil, err
		}
	} else {
		db.DataBagItems[dbi_id] = db_item
	}
	err = db.Save()
	if err != nil {
		return nil, err
	}
	indexer.IndexObj(db_item)
	return db_item, nil
}

func (db *DataBag) DeleteDBItem(db_item_name string) error {
	if config.Config.UseMySQL {
		dbi, err := db.GetDBItem(db_item_name)
		if err != nil {
			return err
		}
		err = dbi.deleteDBItemMySQL()
		if err != nil {
			return err
		}
	} else {
		delete(db.DataBagItems, db_item_name)
	}
	err := db.Save()
	if err != nil {
		return err
	}
	indexer.DeleteItemFromCollection(db.Name, db_item_name)
	return nil
}

func (db *DataBag) GetDBItem(db_item_name string) (*DataBagItem, error) {
	if config.Config.UseMySQL {
		dbi, err := db.getDBItemMySQL(db_item_name)
		if err == sql.ErrNoRows {
			err = fmt.Errorf("data bag item %s in %s not found", db_item_name, db.Name)
		}
		return dbi, err
	} else {
		dbi, ok := db.DataBagItems[db_item_name]
		if !ok {
			err := fmt.Errorf("data bag item %s in %s not found", db_item_name, db.Name)
			return nil, err
		}
		return dbi, nil
	}
}

func (db *DataBag) AllDBItems() (map[string]*DataBagItem, error) {
	if config.Config.UseMySQL {
		return db.allDBItemsMySQL()
	} else {
		return db.DataBagItems, nil
	}
}

func (db *DataBag) ListDBItems() []string {
	if config.Config.UseMySQL {
		return db.listDBItemsMySQL()
	} else {
		dbis := make([]string, len(db.DataBagItems))
		n := 0
		for k := range db.DataBagItems {
			dbis[n] = k
			n++
		}
		return dbis
	}
}

func (db *DataBag) NumDBItems() int {
	if config.Config.UseMySQL {
		return db.numDBItemsMySQL()
	} else {
		return len(db.DataBagItems)
	}
}

func (db *DataBag) fullDBItemName(db_item_name string) string {
	return fmt.Sprintf("data_bag_item_%s_%s", db.Name, db_item_name)
}

// Extract the data bag item's raw data from the request saving it to the 
// server.
func RawDataBagJson (data io.ReadCloser) map[string]interface{} {
	raw_dbag_item := make(map[string]interface{})
	json.NewDecoder(data).Decode(&raw_dbag_item)
	var raw_data map[string]interface{}

	/* The way data can come from knife may
	 * not be entirely consistent. Use 
	 * raw data from the json hash if we
	 * have it, otherwise assume it's just
	 * the raw data without the other chef
	 * stuff added. */

	if _, ok := raw_dbag_item["raw_data"]; ok {
		raw_data = raw_dbag_item["raw_data"].(map[string]interface{})
	} else {
		raw_data = raw_dbag_item
	}
	return raw_data
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
func (dbi *DataBagItem) DocId() string {
	switch did := dbi.RawData["id"].(type) {
		case string:
			return did
		default:
			d := strings.Replace(dbi.Name, dbi.DataBagName, "", 1)
			return d
	}
}

func (dbi *DataBagItem) Index() string {
	return dbi.DataBagName
}

func (dbi *DataBagItem) Flatten() []string {
	flatten := make(map[string]interface{})
	for key, v := range dbi.RawData {
		subExpand := util.DeepMerge(key, v)
		for k, u := range subExpand {
			flatten[k] = u
		}
	}
	indexified := util.Indexify(flatten)
	return indexified
}
