/* Data bags! */

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

// Data bags provide a convenient way to store arbitrary data on the server.
package data_bag

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"fmt"
	"encoding/json"
	"io"
	"net/http"
)

type DataBag struct {
	Name string
	DataBagItems map[string]DataBagItem
}

type DataBagItem struct {
	Name string `json:"name"`
	ChefType string `json:"chef_type"`
	JsonClass string `json:"json_class"`
	DataBagName string `json:"data_bag"`
	RawData map[string]interface{} `json:"raw_data"`
}

/* Data bag functions and methods */

func New(name string) (*DataBag, util.Gerror){
	ds := data_store.New()
	if _, found := ds.Get("data_bag", name); found {
		err := util.Errorf("Data bag %s already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if err := validateDataBagName(name, false); err != nil {
		return nil, err
	}
	dbi_map := make(map[string]DataBagItem)
	data_bag := &DataBag{
		Name: name,
		DataBagItems: dbi_map,
	}
	return data_bag, nil
}

func Get(db_name string) (*DataBag, error){
	ds := data_store.New()
	data_bag, found := ds.Get("data_bag", db_name)
	if !found {
		err := fmt.Errorf("Cannot load data bag %s", db_name)
		return nil, err
	}
	return data_bag.(*DataBag), nil
}

func (db *DataBag) Save() error {
	ds := data_store.New()
	ds.Set("data_bag", db.Name, db)
	return nil
}

func (db *DataBag) Delete() error {
	ds := data_store.New()
	ds.Delete("data_bag", db.Name)
	return nil
}

func GetList() []string {
	ds := data_store.New()
	db_list := ds.GetList("data_bag")
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

func (db *DataBag) NewDBItem (raw_dbag_item map[string]interface{}) (*DataBagItem, util.Gerror){
	//dbi_id := raw_dbag_item["id"].(string)
	var dbi_id string
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

	/* Look for an existing dbag item with this name */
	if _, found := db.DataBagItems[dbi_id]; found {
		err := util.Errorf("Item %s in data bag %s already exists.", dbi_id, db.Name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	if err := validateDataBagName(dbi_id, true); err != nil {
		return nil, err
	}
	dbi_full_name := fmt.Sprintf("data_bag_item_%s_%s", db.Name, dbi_id)
	/* But should we store the raw data as a JSON string? */
	dbag_item := DataBagItem{
		Name: dbi_full_name,
		ChefType: "data_bag_item",
		JsonClass: "Chef::DataBagItem",
		DataBagName: db.Name,
		RawData: raw_dbag_item,
	}
	db.DataBagItems[dbi_id] = dbag_item
	/* ? */
	db.Save()
	return &dbag_item, nil
}

func (db *DataBag) UpdateDBItem(dbi_id string, raw_dbag_item map[string]interface{}) (*DataBagItem, error){
	db_item, found := db.DataBagItems[dbi_id]
	if !found {
		err := fmt.Errorf("Item %s in data bag %s does not exist.", dbi_id, db.Name)
		return nil, err
	}
	db_item.RawData = raw_dbag_item
	db.DataBagItems[dbi_id] = db_item
	db.Save()
	return &db_item, nil
}

func (db *DataBag) DeleteDBItem(db_item_name string) error {
	delete(db.DataBagItems, db_item_name)
	db.Save()
	return nil
}

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
