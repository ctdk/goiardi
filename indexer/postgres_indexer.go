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
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
)

type PostgresIndex struct {

}

func (p *PostgresIndex) Initialize() error {
	// check if the default indexes exist yet, and if not create them
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
	_, err = tx.Exec("SELECT goiardi.delete_search_item($1, $2)", idxName, doc, 1)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) SaveItem(obj Indexable) error {

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
