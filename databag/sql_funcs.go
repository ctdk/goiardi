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

package databag

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"log"
	"strings"
)

// Functions for finding, saving, etc. data bags with an SQL database.

func checkForDataBagSQL(dbhandle datastore.Dbhandle, name string) (bool, error) {
	_, err := datastore.CheckForOne(dbhandle, "data_bags", name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

func getDataBagSQL(name string) (*DataBag, error) {
	dataBag := new(DataBag)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT id, name FROM data_bags WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT id, name FROM goiardi.data_bags WHERE name = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(name).Scan(&dataBag.id, &dataBag.Name)
	if err != nil {
		return nil, err
	}
	return dataBag, nil
}

func (dbi *DataBagItem) fillDBItemFromSQL(row datastore.ResRow) error {
	var rawb []byte
	err := row.Scan(&dbi.id, &dbi.dataBagID, &dbi.Name, &dbi.origName, &dbi.DataBagName, &rawb)
	if err != nil {
		return err
	}
	dbi.ChefType = "data_bag_item"
	dbi.JSONClass = "Chef::DataBagItem"
	err = datastore.DecodeBlob(rawb, &dbi.RawData)
	if err != nil {
		return err
	}
	datastore.ChkNilArray(dbi)
	return nil
}

func (db *DataBag) getDBItemSQL(dbItemName string) (*DataBagItem, error) {
	dbi := new(DataBagItem)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.orig_name = ? AND dbi.data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM goiardi.data_bag_items dbi JOIN goiardi.data_bags db on dbi.data_bag_id = db.id WHERE dbi.orig_name = $1 AND dbi.data_bag_id = $2"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(dbItemName, db.id)
	err = dbi.fillDBItemFromSQL(row)
	if err != nil {
		return nil, err
	}
	return dbi, nil
}

func (db *DataBag) checkDBItemSQL(dbItemName string) (bool, error) {
	var found bool
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT COUNT(dbi.id) FROM data_bag_items dbi JOIN data_bags db ON dbi.data_bag_id = db.id WHERE dbi.orig_name = ? AND dbi.data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT COUNT(dbi.id) FROM goiardi.data_bag_items dbi JOIN goiardi.data_bags db on dbi.data_bag_id = db.id WHERE dbi.orig_name = $1 AND dbi.data_bag_id = $2"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var c int
	err = stmt.QueryRow(dbItemName, db.id).Scan(&c)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if c != 0 {
		found = true
	}
	return found, nil
}

func (db *DataBag) getMultiDBItemSQL(dbItemNames []string) ([]*DataBagItem, error) {
	var sqlStmt string
	bind := make([]string, len(dbItemNames))

	if config.Config.UseMySQL {
		for i := range dbItemNames {
			bind[i] = "?"
		}
		sqlStmt = fmt.Sprintf("SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = ? AND dbi.orig_name IN (%s)", strings.Join(bind, ", "))
	} else if config.Config.UsePostgreSQL {
		for i := range dbItemNames {
			bind[i] = fmt.Sprintf("$%d", i+2)
		}
		sqlStmt = fmt.Sprintf("SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM goiardi.data_bag_items dbi JOIN goiardi.data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = $1 AND dbi.orig_name IN (%s)", strings.Join(bind, ", "))
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	nameArgs := make([]interface{}, len(dbItemNames)+1)
	nameArgs[0] = db.id
	for i, v := range dbItemNames {
		nameArgs[i+1] = v
	}
	rows, err := stmt.Query(nameArgs...)
	if err != nil {
		return nil, err
	}
	dbis := make([]*DataBagItem, 0, len(dbItemNames))
	for rows.Next() {
		d := new(DataBagItem)
		err = d.fillDBItemFromSQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		dbis = append(dbis, d)
	}

	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return dbis, nil
}

func (dbi *DataBagItem) updateDBItemSQL() error {
	rawb, rawerr := datastore.EncodeBlob(&dbi.RawData)
	if rawerr != nil {
		return rawerr
	}
	tx, err := datastore.Dbh.Begin()
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
			err = fmt.Errorf("updating data bag item %s in data bag %s had an error '%s', and then rolling back the transaction gave another error '%s'", dbi.origName, dbi.DataBagName, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func (dbi *DataBagItem) deleteDBItemSQL() error {
	tx, err := datastore.Dbh.Begin()
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
			err = fmt.Errorf("deleting data bag item %s in data bag %s had an error '%s', and then rolling back the transaction gave another error '%s'", dbi.origName, dbi.DataBagName, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func (db *DataBag) allDBItemsSQL() (map[string]*DataBagItem, error) {
	dbis := make(map[string]*DataBagItem)
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM data_bag_items dbi JOIN data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT dbi.id, dbi.data_bag_id, dbi.name, dbi.orig_name, db.name, dbi.raw_data FROM goiardi.data_bag_items dbi JOIN goiardi.data_bags db on dbi.data_bag_id = db.id WHERE dbi.data_bag_id = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(db.id)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return dbis, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		dbi := new(DataBagItem)
		err = dbi.fillDBItemFromSQL(rows)
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
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var dbiCount int
	err = stmt.QueryRow(db.id).Scan(&dbiCount)
	if err != nil {
		if err == sql.ErrNoRows {
			dbiCount = 0
		} else {
			log.Fatal(err)
		}
	}
	return dbiCount
}

func (db *DataBag) listDBItemsSQL() []string {
	var dbiList []string
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT orig_name FROM data_bag_items WHERE data_bag_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT orig_name FROM goiardi.data_bag_items WHERE data_bag_id = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query(db.id)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return dbiList
	}
	for rows.Next() {
		var dbiName string
		err = rows.Scan(&dbiName)
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}
		dbiList = append(dbiList, dbiName)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	return dbiList
}

func (db *DataBag) deleteSQL() error {
	tx, err := datastore.Dbh.Begin()
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
			err = fmt.Errorf("deleting data bag items for data bag %s had an error '%s', and then rolling back the transaction gave another error '%s'", db.Name, err.Error(), terr.Error())
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
			err = fmt.Errorf("deleting data bag %s had an error '%s', and then rolling back the transaction gave another error '%s'", db.Name, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func getListSQL() []string {
	var dbList []string
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT name FROM data_bags"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT name FROM goiardi.data_bags"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		return dbList
	}
	for rows.Next() {
		var dbName string
		err = rows.Scan(&dbName)
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}
		dbList = append(dbList, dbName)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	return dbList
}
func allDataBagsSQL() []*DataBag {
	var dbags []*DataBag
	var sqlStatement string
	if config.Config.UseMySQL {
		sqlStatement = "SELECT id, name FROM data_bags"
	} else if config.Config.UsePostgreSQL {
		sqlStatement = "SELECT id, name FROM goiardi.data_bags"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
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
		dataBag := new(DataBag)
		err = rows.Scan(&dataBag.id, &dataBag.Name)
		if err != nil {
			log.Fatal(err)
		}
		dataBag.DataBagItems, err = dataBag.allDBItemsSQL()
		if err != nil {
			log.Fatal(err)
		}
		dbags = append(dbags, dataBag)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return dbags
}
