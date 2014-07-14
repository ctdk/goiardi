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

package node

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/lib/pq"
)

func (n *Node) savePostgreSQL(tx datastore.Dbhandle, rlb, aab, nab, dab, oab []byte) error {
	_, err := tx.Exec("SELECT goiardi.merge_nodes($1, $2, $3, $4, $5, $6, $7)", n.Name, n.ChefEnvironment, rlb, aab, nab, dab, oab)
	if err != nil {
		return err
	}
	return nil
}

func (ns *NodeStatus) updateNodeStatusPostgreSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("SELECT goiardi.insert_node_status($1, $2)", ns.Node.Name, ns.Status)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (ns *NodeStatus) fillNodeStatusFromPostgreSQL(row datastore.ResRow) error {
	var ua pq.NullTime
	err := row.Scan(&ns.Status, &ua)
	if err != nil {
		return nil
	}
	if ua.Valid {
		ns.UpdatedAt = ua.Time
	}
	return nil
}
