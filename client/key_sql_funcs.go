/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// SQL functions for client keys

package client

import (
	"database/sql"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func (k *Key) fillKeyFromSQL(row datastore.ResRow) error {
	if err := row.Scan(&k.Name, &k.PublicKey, &k.ExpirationDate, &k.clientId, &k.id); err != nil {
		return err
	}
	return nil
}

func (c *Client) getKeySQL(name string) (*Key, error) {
	key := new(Key)

	sqlStatement := "SELECT name, public_key, expiration_date, client_id, id FROM goiardi.client_keys WHERE client_id = $1 AND name = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(c.GetId(), name)
	err = key.fillKeyFromSQL(row)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (c *Client) getAllKeysSQL() (map[string]*Key, error) {
	keys := make(map[string]*Key)
	sqlStatement := "SELECT name, public_key, expiration_date, client_id, id FROM goiardi.client_keys WHERE client_id = $1"
	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(c.GetId())
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return keys, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		k := new(Key)
		err = k.fillKeyFromSQL(rows)
		if err != nil {
			return nil, err
		}
		keys[k.Name] = k
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (c *Client) deleteKeySQL(name string) util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}

	_, err = tx.Exec("DELETE FROM goiardi.client_keys WHERE client_id = $1 AND name = $2", c.GetId(), name)

	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}

func (c *Client) deleteAllKeysSQL() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}

	_, err = tx.Exec("DELETE FROM goiardi.client_keys WHERE client_id = $1", c.GetId())

	if err != nil {
		tx.Rollback()
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}
