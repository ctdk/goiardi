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

func checkForDataBagMySQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
	_, err := data_store.CheckForOne(dbhandle, "data_bags", name)
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
	err := row.Scan(&dbi.id, &dbi.data_bag_id, &dbi.Name, &dbi.origName, &dbi.DataBagName, &rawb)
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
	stmt, err := data_store.Dbh.Prepare("SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.orig_name = ? AND dbi.data_bag_id = ?")
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

func (db *DataBag) newDBItemMySQL(dbi_id string, raw_dbag_item map[string]interface{}) (*DataBagItem, error){
	rawb, rawerr := data_store.EncodeBlob(raw_dbag_item)
	if rawerr != nil {
		return nil, rawerr
	}

	dbi := &DataBagItem{
		Name: db.fullDBItemName(dbi_id),
		ChefType: "data_bag_item",
		JsonClass: "Chef::DataBagItem",
		DataBagName: db.Name,
		RawData: raw_dbag_item,
		origName: dbi_id,
		data_bag_id: db.id,
	}
	
	tx, err := data_store.Dbh.Begin()
	// make sure this data bag didn't go away while we were doing something
	// else
	found, ferr := checkForDataBagMySQL(tx, db.Name)
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

	return dbi, nil
}

func (dbi *DataBagItem) updateDBItemMySQL() error {
	rawb, rawerr := data_store.EncodeBlob(raw_dbag_item)
	if rawerr != nil {
		return nil, rawerr
	}
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE data_bag_items SET raw_data = ?, updated_at = NOW() WHERE id = ?", rawb, dbi.id)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("updating data bag item %s in data bag %s had an error '%s', and then rolling back the transaction gave another erorr '%s'", dbi.origName, dbi.DataBagName, err.Error(), terr.Error())
		}
		return err
	}
}

func (dbi *DataBagItem) deleteDBItemMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM data_bag_items WHERE id = ?", dbi.id)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting data bag item %s in data bag %s had an error '%s', and then rolling back the transaction gave another erorr '%s'", dbi.origName, dbi.DataBagName, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func (db *DataBag) allDBItemsMySQL()(map[string]*DataBagItem, error) {
	dbis := make(map[string]*DataBagItem)
	stmt, err := data_store.Dbh.Prepare("SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = ?")
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
	stmt, err := data_store.Dbh.Prepare("SELECT orig_name FROM data_bag_items WHERE data_bag_id = ?")
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

func (db *DataBag) deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM data_bag_items WHERE data_bag_id = ?", db.id)
	if err != nil && err != sql.ErrNoRows {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting data bag items for data bag %s had an error '%s', and then rolling back the transaction gave another erorr '%s'", db.Name, err.Error(), terr.Error())
		}
		return err
	}
	_, err = tx.Exec("DELETE FROM data_bags WHERE id = ?", db.id)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting data bag %s had an error '%s', and then rolling back the transaction gave another erorr '%s'", db.Name, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func (db *DataBag) saveMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	found, ferr := checkForDataBagMySQL(tx, db.Name)
	if err == nil {
		_, err = tx.Exec("UPDATE data_bags SET updated_at = NOW() WHERE id = ?", db.id)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}
		res, rerr := tx.Exec("INSERT INTO data_bags (name, created_at, updated_at) VALUES (?, NOW(), NOW())", db.Name)
		if rerr != nil {
			tx.Rollback()
			return rerr
		}
		db_id, err := res.LastInsertId()
		db.id = int32(db_id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func getListMySQL() []string {
	db_list := make([]string, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT name FROM data_bags")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return dbi_list
	}
	for rows.Next() {
		var db_name string
		err = rows.Scan(&db_name)
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}
		db_list = append(db_list, db_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	return db_list
}
