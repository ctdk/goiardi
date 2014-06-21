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
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/data_store"
	"database/sql"
	"fmt"
	"log"
)

// Functions for finding, saving, etc. data bags with an SQL database.

func checkForDataBagSQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
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

func getDataBagSQL(name string) (*DataBag, error) {
	data_bag := new(DataBag)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT id, name FROM data_bags WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT id, name FROM goiardi.data_bags WHERE name = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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
	if err != nil {
		return err
	}
	dbi.ChefType = "data_bag_item"
	dbi.JsonClass = "Chef::DataBagItem"
	err = data_store.DecodeBlob(rawb, &dbi.RawData)
	if err != nil {
		return err
	}
	data_store.ChkNilArray(dbi)
	return nil
}

func (db *DataBag) getDBItemSQL(db_item_name string) (*DataBagItem, error) {
	dbi := new(DataBagItem)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.orig_name = ? AND dbi.data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM goiardi.data_bag_items dbi JOIN goiardi.data_bags db on dbi.data_bag_id = db.id WHERE dbi.orig_name = $1 AND dbi.data_bag_id = $2"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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

func (dbi *DataBagItem) updateDBItemSQL() error {
	rawb, rawerr := data_store.EncodeBlob(&dbi.RawData)
	if rawerr != nil {
		return rawerr
	}
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	if config.Config.UseMySQL {
		_, err = tx.Exec("UPDATE data_bag_items SET raw_data = ?, updated_at = NOW() WHERE id = ?", rawb, dbi.id)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("UPDATE goiardi.data_bag_items SET raw_data = $1, updated_at = NOW() WHERE id = $2", rawb, dbi.id)
	}
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("updating data bag item %s in data bag %s had an error '%s', and then rolling back the transaction gave another erorr '%s'", dbi.origName, dbi.DataBagName, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func (dbi *DataBagItem) deleteDBItemSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM data_bag_items WHERE id = ?", dbi.id)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("DELETE FROM goiardi.data_bag_items WHERE id = $1", dbi.id)
	}
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

func (db *DataBag) allDBItemsSQL()(map[string]*DataBagItem, error) {
	dbis := make(map[string]*DataBagItem)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM goiardi.data_bag_items dbi JOIN goiardi.data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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
		dbis[dbi.origName] = dbi
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return dbis, nil
}

func (db *DataBag) numDBItemsSQL() int {
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT count(*) FROM data_bag_items WHERE data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT count(*) FROM goiardi.data_bag_items WHERE data_bag_id = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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

func (db *DataBag) listDBItemsSQL() []string {
	dbi_list := make([]string, 0)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT orig_name FROM data_bag_items WHERE data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT orig_name FROM goiardi.data_bag_items WHERE data_bag_id = $1"
	}
	stmt, err := data_store.Dbh.Prepare(sqlStatement)
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

func (db *DataBag) deleteSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM data_bag_items WHERE data_bag_id = ?", db.id)
	} else if config.Config.UsePostgreSQL {
		_, err = tx.Exec("DELETE FROM goiardi.data_bag_items WHERE data_bag_id = $1", db.id)
	}
	if err != nil && err != sql.ErrNoRows {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting data bag items for data bag %s had an error '%s', and then rolling back the transaction gave another erorr '%s'", db.Name, err.Error(), terr.Error())
		}
		return err
	}
	if config.Config.UseMySQL {
		_, err = tx.Exec("DELETE FROM data_bags WHERE id = ?", db.id)
	} else if config.Config.UsePostgreSQL {
			_, err = tx.Exec("DELETE FROM goiardi.data_bags WHERE id = $1", db.id)
	}
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

func getListSQL() []string {
	db_list := make([]string, 0)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM data_bags"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.data_bags"
	}

	stmt, err := data_store.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return db_list
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

func allDataBagsSQL() []*DataBag {
	dbags := make([]*DataBag, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT id, name FROM data_bags")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return dbags
	}
	for rows.Next() {
		data_bag := new(DataBag)
		err = rows.Scan(&data_bag.id, &data_bag.Name)
		if err != nil {
			log.Fatal(err)
		}
		data_bag.DataBagItems, err = data_bag.allDBItemsSQL()
		if err != nil {
			log.Fatal(err)
		}
		dbags = append(dbags, data_bag)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return dbags
}
