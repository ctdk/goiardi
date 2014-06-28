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

package data_bag

import (
	"github.com/ctdk/goiardi/data_store"
)

// PostgreSQL-specific functions for data bags & data bag items.

func (db *DataBag) newDBItemPostgreSQL(dbi_id string, raw_dbag_item map[string]interface{}) (*DataBagItem, error) {
	rawb, rawerr := data_store.EncodeBlob(&raw_dbag_item)
	if rawerr != nil {
		return nil, rawerr
	}

	dbi := &DataBagItem{
		Name:        db.fullDBItemName(dbi_id),
		ChefType:    "data_bag_item",
		JsonClass:   "Chef::DataBagItem",
		DataBagName: db.Name,
		RawData:     raw_dbag_item,
		origName:    dbi_id,
		data_bag_id: db.id,
	}

	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return nil, err
	}

	// make sure this data bag didn't go away while we were doing something
	// else
	err = tx.QueryRow("SELECT goiardi.insert_dbi($1, $2, $3, $4, $5)", db.Name, dbi.Name, dbi.origName, db.id, rawb).Scan(&dbi.id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()

	return dbi, nil
}

func (db *DataBag) savePostgreSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}

	err = tx.QueryRow("SELECT goiardi.merge_data_bags($1)", db.Name).Scan(&db.id)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}
