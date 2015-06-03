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

package indexer

import (
	"fmt"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"github.com/lib/pq"
)

type PostgresIndex struct {
}

func (p *PostgresIndex) Initialize() error {
	// check if the default indexes exist yet, and if not create them
	var c int
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	// organization_id will obviously not always be 1
	err = tx.QueryRow("SELECT count(*) FROM goiardi.search_collections WHERE organization_id = $1 AND name IN ('node', 'client', 'environment', 'role')", 1).Scan(&c)
	if err != nil {
		tx.Rollback()
		return err
	}
	if c != 0 {
		if c != 4 {
			err = fmt.Errorf("Aiiie! We were going to initialize the database, but while we expected there to be either 0 or 4 of the basic search types to be in place, there were only %d. Aborting.", c)
			tx.Rollback()
			return err
		}
		// otherwise everything's good.
	} else {
		sqlStmt := "INSERT INTO goiardi.search_collections (name, organization_id) VALUES ('client', $1), ('environment', $1), ('node', $1), ('role', $1)"
		_, err = tx.Exec(sqlStmt, 1)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) CreateCollection(col string) error {
	sqlStmt := "INSERT INTO goiardi.search_collections (name, organization_id) VALUES ($1, $2)"
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStmt, col, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) DeleteCollection(col string) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("SELECT goiardi.delete_search_collection($1, $2)", col, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) DeleteItem(idxName string, doc string) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("SELECT goiardi.delete_search_item($1, $2, $3)", idxName, doc, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) SaveItem(obj Indexable) error {
	flat := obj.Flatten()
	itemName := obj.DocID()
	collectionName := obj.Index()
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	var scID int32
	err = tx.QueryRow("SELECT id FROM goiardi.search_collections WHERE organization_id = $1 AND name = $2", 1, collectionName).Scan(&scID)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec("SELECT goiardi.delete_search_item($1, $2, $3)", collectionName, itemName, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, _ = tx.Exec("SET search_path TO goiardi")
	stmt, err := tx.Prepare(pq.CopyIn("search_items", "organization_id", "search_collection_id", "item_name", "value", "path"))
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for k, v := range flat {
		k = util.PgSearchKey(k)
		// will the values need escaped like in file search?
		switch v := v.(type) {
		case string:
			v = util.IndexEscapeStr(v)
			_, err = stmt.Exec(1, scID, itemName, v, k)
			if err != nil {
				tx.Rollback()
				return err
			}
		case []string:
			for _, w := range v {
				w = util.IndexEscapeStr(w)
				_, err = stmt.Exec(1, scID, itemName, w, k)
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		default:
			err = fmt.Errorf("pg search should have never been able to reach this state. Key %s had a value %v of type %T", k, v, v)
			tx.Rollback()
			return err
		}
	}
	_, err = stmt.Exec()
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (p *PostgresIndex) Endpoints() ([]string, error) {
	sqlStmt := "SELECT ARRAY_AGG(name) FROM goiardi.search_collections WHERE organization_id = $1"
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var endpoints util.StringSlice
	err = stmt.QueryRow(1).Scan(&endpoints)
	if err != nil {
		return nil, err
	}

	return endpoints, nil
}

func (p *PostgresIndex) Clear() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	sqlStmt := "DELETE FROM goiardi.search_items WHERE organization_id = $1"
	_, err = tx.Exec(sqlStmt, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	sqlStmt = "DELETE FROM goiardi.search_collections WHERE organization_id = $1"
	_, err = tx.Exec(sqlStmt, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	sqlStmt = "INSERT INTO goiardi.search_collections (name, organization_id) VALUES ('client', $1), ('environment', $1), ('node', $1), ('role', $1)"
	_, err = tx.Exec(sqlStmt, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	return nil
}
