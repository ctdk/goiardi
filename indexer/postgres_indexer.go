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

package indexer

import (
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"github.com/lib/pq"
	"sort"
	"strings"
)

type PostgresIndex struct {
}

func (p *PostgresIndex) Initialize(org IndexerOrg) error {
	// check if the default indexes exist yet, and if not create them
	var c int
	var schemaExists bool

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	defaultOrgSchema := org.SearchSchemaName()

	// Check if the default org search schema exists.
	err = tx.QueryRow("SELECT exists(SELECT schema_name FROM information_schema.schemata WHERE schema_name = $1)", defaultOrgSchema).Scan(&schemaExists)
	if err != nil {
		tx.Rollback()
		return err
	}

	// If it doesn't, it needs to be created. This does duplicate an
	// internal method inside organizations, but it can't really be avoided
	// sadly.

	if !schemaExists {
		_, err = tx.Exec("SELECT goiardi.clone_schema($1, $2)", util.BaseSearchSchema, defaultOrgSchema)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s.search_collections WHERE name IN ('node', 'client', 'environment', 'role')", defaultOrgSchema)).Scan(&c)
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
		sqlStmt := fmt.Sprintf("INSERT INTO %s.search_collections (name, organization_id) VALUES ('client', $1), ('environment', $1), ('node', $1), ('role', $1)", defaultOrgSchema)
		_, err = tx.Exec(sqlStmt, org.GetId())
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) CreateOrgDex(org IndexerOrg) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	schemaName := org.SearchSchemaName()
	_, err = tx.Exec("SELECT goiardi.clone_schema($1, $2)", util.BaseSearchSchema, schemaName)
	if err != nil {
		tx.Rollback()
		return err
	}

	sqlStmt := fmt.Sprintf("INSERT INTO %s.search_collections (name, organization_id) VALUES ('client', $1), ('environment', $1), ('node', $1), ('role', $1)", schemaName)
	if _, err = tx.Exec(sqlStmt, org.GetId()); err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (p *PostgresIndex) DeleteOrgDex(org IndexerOrg) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("CALL goiardi.drop_search_schema($1)", org.SearchSchemaName())
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (p *PostgresIndex) CreateCollection(org IndexerOrg, col string) error {
	sqlStmt := fmt.Sprintf("INSERT INTO %s.search_collections (name, organization_id) VALUES ($1, $2)", org.SearchSchemaName())
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStmt, col, org.GetId())
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) CreateNewCollection(org IndexerOrg, col string) error {
	return p.CreateCollection(org, col)
}

func (p *PostgresIndex) DeleteCollection(org IndexerOrg, col string) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(fmt.Sprintf("SELECT %s.delete_search_collection($1, $2)", org.SearchSchemaName()), col, org.GetId())
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) DeleteItem(org IndexerOrg, idxName string, doc string) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(fmt.Sprintf("SELECT %s.delete_search_item($1, $2, $3)", org.SearchSchemaName()), idxName, doc, org.GetId())
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (p *PostgresIndex) SaveItem(org IndexerOrg, obj Indexable) error {
	flat := obj.Flatten()
	itemName := obj.DocID()
	collectionName := obj.Index()
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	orgSchema := org.SearchSchemaName()
	var scID int32
	err = tx.QueryRow(fmt.Sprintf("SELECT id FROM %s.search_collections WHERE organization_id = $1 AND name = $2", orgSchema), org.GetId(), collectionName).Scan(&scID)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(fmt.Sprintf("SELECT %s.delete_search_item($1, $2, $3)", orgSchema), collectionName, itemName, org.GetId())
	if err != nil {
		tx.Rollback()
		return err
	}
	_, _ = tx.Exec(fmt.Sprintf("SET search_path TO %s", orgSchema))
	stmt, err := tx.Prepare(pq.CopyIn("search_items", "organization_id", "search_collection_id", "item_name", "value", "path"))
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	maxValLen := config.Config.IndexValTrim
	for k, v := range flat {
		k = util.PgSearchKey(k)
		// will the values need escaped like in file search?
		switch v := v.(type) {
		case string:
			v = util.TrimStringMax(v, maxValLen)
			v = util.IndexEscapeStr(v)
			// try it with newlines too
			v = strings.Replace(v, "\n", "\\n", -1)
			_, err = stmt.Exec(org.GetId(), scID, itemName, v, k)
			if err != nil {
				tx.Rollback()
				return err
			}
		case []string:
			// remove dupes from slices of strings like we're doing
			// now with the trie index, both to reduce ambiguity and
			// to maybe make the indexes just a little bit smaller
			sort.Strings(v)
			v = util.RemoveDupStrings(v)
			for _, w := range v {
				w = util.TrimStringMax(w, maxValLen)
				w = util.IndexEscapeStr(w)
				w = strings.Replace(w, "\n", "\\n", -1)
				_, err = stmt.Exec(org.GetId(), scID, itemName, w, k)
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

func (p *PostgresIndex) Endpoints(org IndexerOrg) ([]string, error) {
	sqlStmt := fmt.Sprintf("SELECT ARRAY_AGG(name) FROM %s.search_collections WHERE organization_id = $1", org.SearchSchemaName())
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

func (p *PostgresIndex) Clear(org IndexerOrg) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	orgSchema := org.SearchSchemaName()

	// Ooooh. Now, rather than doing a whole dance with locking tables and
	// such and such, we can just torpedo the whole schema and rebuild it.
	if _, err = tx.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", orgSchema)); err != nil {
		tx.Rollback() // this might not actually work
		return err
	}

	// Rebuild yon schema
	if _, err = tx.Exec("SELECT goiardi.clone_schema($1, $2)", util.BaseSearchSchema, orgSchema); err != nil {
		tx.Rollback()
		return err
	}

	sqlStmt := fmt.Sprintf("INSERT INTO %s.search_collections (name, organization_id) VALUES ('client', $1), ('environment', $1), ('node', $1), ('role', $1)", orgSchema)
	_, err = tx.Exec(sqlStmt, org.GetId())
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	return nil
}

func (p *PostgresIndex) OrgList() []string {
	return nil
}
