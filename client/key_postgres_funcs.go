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

// Postegres-specific SQL functions for client keys (for ease later should we
// decide to add sqlite).

package client

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
)

func (c *Client) saveKeyPostgreSQL(key *Key) util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return util.CastErr(err)
	}
	err = tx.QueryRow("SELECT goiardi.merge_client_keys($1, $2, $3, $4)", key.Name, key.PublicKey, key.ExpirationDate, c.GetId()).Scan(&key.id)
	if err != nil {
		tx.Rollback()
		return util.CastErr(err)
	}
	tx.Commit()
	return nil
}
