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

package databag

import (
	"fmt"
	"github.com/ctdk/goiardi/datastore"
)

// MySQL-specific functions for data bags & data bag items.

func (db *DataBag) newDBItemMySQL(dbiID string, rawDbagItem map[string]interface{}) (*DataBagItem, error) {
	rawb, rawerr := datastore.EncodeBlob(&rawDbagItem)
	if rawerr != nil {
		return nil, rawerr
	}

	dbi := &DataBagItem{
		Name:        db.fullDBItemName(dbiID),
		ChefType:    "data_bag_item",
		JSONClass:   "Chef::DataBagItem",
		DataBagName: db.Name,
		RawData:     rawDbagItem,
		origName:    dbiID,
		dataBagID: db.id,
	}

	tx, err := datastore.Dbh.Begin()
	// make sure this data bag didn't go away while we were doing something
	// else
	found, ferr := checkForDataBagSQL(tx, db.Name)
	if ferr != nil {
		tx.Rollback()
		return nil, err
	} else if !found {
		tx.Rollback()
		err = fmt.Errorf("aiiiie! The data bag %s was deleted from the db while we were doing something else", db.Name)
		return nil, err
	}
	res, err := tx.Exec("INSERT INTO data_bag_items (name, orig_name, data_bag_id, raw_data, created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW())", dbi.Name, dbi.origName, db.id, rawb)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	did, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	dbi.id = int32(did)
	tx.Commit()

	return dbi, nil
}

func (db *DataBag) saveMySQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	res, rerr := tx.Exec("INSERT INTO data_bags (name, created_at, updated_at) VALUES (?, NOW(), NOW()) ON DUPLICATE KEY UPDATE updated_at = NOW()", db.Name)
	if rerr != nil {
		tx.Rollback()
		return rerr
	}
	if db.id == 0 {
		dbID, err := res.LastInsertId()
		db.id = int32(dbID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}
