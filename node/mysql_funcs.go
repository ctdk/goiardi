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

/* MySQL specific functions for nodes */

package node

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/go-sql-driver/mysql"
)

func (n *Node) saveMySQL(tx datastore.Dbhandle, rlb, aab, nab, dab, oab []byte) error {
	_, err := tx.Exec("INSERT INTO nodes (name, chef_environment, run_list, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE chef_environment = ?, run_list = ?, automatic_attr = ?, normal_attr = ?, default_attr = ?, override_attr = ?, updated_at = NOW()", n.Name, n.ChefEnvironment, rlb, aab, nab, dab, oab, n.ChefEnvironment, rlb, aab, nab, dab, oab)
	if err != nil {
		return err
	}
	return nil
}

func (ns *NodeStatus) updateNodeStatusMySQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO node_statuses (node_id, status, updated_at) SELECT id, ?, NOW() FROM nodes WHERE name = ?", ns.Status, ns.Node.Name)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (ns *NodeStatus) fillNodeStatusFromMySQL(row datastore.ResRow) error {
	var ua mysql.NullTime
	err := row.Scan(&ns.Status, &ua)
	if err != nil {
		return nil
	}
	if ua.Valid {
		ns.UpdatedAt = ua.Time
	}
	return nil
}
