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
	"database/sql"
	"fmt"
)

// Functions for finding, saving, etc. data bags with a MySQL database.

func checkForDataBagMySQL(name string) (bool, error) {
	_, err := data_store.CheckForOne(data_store.Dbh, "data_bags", name)
	if err == nil {
		return true, nil
	} else {
		if err != sql.ErrNoRows {
			return false, err
		} else {
			return false, nil
		}
	}
}

func getDataBagMySQL(name string) (*DataBag, error) {
	data_bag := new(DataBag)
	stmt, err := data_store.Dbh.Prepare("SELECT id, name FROM data_bags WHERE name = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(name).Scan(&data_bag.id, &data_bag.Name)
	if err != nil {
		return nil, err
	}
	return data_bag, nil
}

func (dbi *DataBagItem) fillDBItemFromMySQL(row data_store.ResRow) error {
	var rawb []byte
	err := row.Scan(&dbi.id, &dbi.data_bag_id, &dbi.Name, &dbi.DataBagName, &rawb)
	dbi.ChefType = "data_bag_item"
	dbi.JsonClass = "Chef::DataBagItem"
	var q interface{}
	q, err = data_store.DecodeBlob(rawb, dbi.RawData)
	if err != nil {
		return err
	}
	dbi.RawData = q.(map[string]interface{})
	data_store.CheckNilArray(dbi)
	return nil
}

func (db *DataBag) getDBItemMySQL(db_item_name string) (*DataBagItem, error) {
	dbi := new(DataBagItem)
	stmt, err := data_store.Dbh.Prepare("SELECT dbi.id, dbi.data_bag_id, dbi.name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.name = ? AND dbi.data_bag_id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(db_item_name, db.id)
	err = dbi.fillDBItemFromMySQL(row)
	if err != nil {
		return nil, err
	}
	return dbi, nil
}

func (db *DataBag) allDBItemsMySQL()(map[string]*DataBagItem, error) {
	dbis := make(map[string]*DataBagItem)
	stmt, err := data_store.Dbh.Prepare("SELECT dbi.id, dbi.data_bag_id, dbi.name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(db.id)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return dbis, nil
		} else {
			return nil, qerr
		}
	}
	for rows.Next() {
		dbi := new(DataBagItem)
		err = dbi.fillDBItemFromMySQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		dbis = append(dbis, dbi)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return dbis, nil
}

func (db *DataBag) numDBItemsMySQL() int {
	stmt, err := data_store.Dbh.Prepare("SELECT count(*) FROM data_bag_items WHERE data_bag_id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var dbi_count int
	err = stmt.QueryRow(db.id).Scan(&dbi_count)
	if err != nil {
		if err == sql.ErrNoRows {
			dbi_count = 0
		} else {
			log.Fatal(err)
		}
	}
	return dbi_count
}

func (db *DataBag) listDBItemsMySQL() []string {
	dbi_list := make([]string, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT name FROM data_bag_items WHERE data_bag_id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query(db.id)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return dbi_list
	}
	for rows.Next() {
		var dbi_name string
		err = rows.Scan(&dbi_name)
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}
		dbi_list = append(dbi_list, dbi_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	return dbi_list
}
