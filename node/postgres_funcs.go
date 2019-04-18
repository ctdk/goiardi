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

package node

import (
	"database/sql"
	"github.com/ctdk/goiardi/datastore"
	"github.com/lib/pq"
	"strings"
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
	var isDown bool
	if ns.Status == "down" {
		isDown = true
	}
	if isDown != ns.Node.isDown {
		_, err = tx.Exec("UPDATE goiardi.nodes SET is_down = $1, updated_at = NOW() WHERE name = $2", isDown, ns.Node.Name)
		if err != nil {
			tx.Rollback()
			return err
		}
		ns.Node.isDown = isDown
	}
	tx.Commit()
	if isDown != ns.Node.isDown {
		ns.Node.Save()
	}
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

func getNodesByStatusPostgreSQL(nodeNames []string, status string) ([]*Node, error) {
	var nodes []*Node
	sqlStmt := "SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM goiardi.node_latest_statuses n WHERE n.status = $1 AND n.name = ANY($2::text[])"
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	nodeStr := "{" + strings.Join(nodeNames, ",") + "}"
	rows, qerr := stmt.Query(status, nodeStr)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nodes, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		no := new(Node)
		err = no.fillNodeFromSQL(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, no)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}
