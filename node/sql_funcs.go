/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

/* Generic SQL functions for nodes */

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"log"
	"strings"
	"time"
)

func checkForNodeSQL(dbhandle datastore.Dbhandle, name string) (bool, error) {
	_, err := datastore.CheckForOne(datastore.Dbh, "nodes", name)
	if err == nil {
		return true, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	return false, nil
}

// Fill in a node from a row returned from the SQL server. Useful for the case
// down the road where an array of objects is needed, but building it with
// a call to GetList(), then repeated calls to Get() sucks with a real db even
// if it's marginally acceptable in in-memory mode.
//
// NB: This does require the query to look like the one in Get().
func (n *Node) fillNodeFromSQL(row datastore.ResRow) error {
	var (
		rl []byte
		aa []byte
		na []byte
		da []byte
		oa []byte
	)
	err := row.Scan(&n.Name, &n.ChefEnvironment, &rl, &aa, &na, &da, &oa)
	if err != nil {
		return err
	}
	n.ChefType = "node"
	n.JSONClass = "Chef::Node"
	err = datastore.DecodeBlob(rl, &n.RunList)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(aa, &n.Automatic)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(na, &n.Normal)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(da, &n.Default)
	if err != nil {
		return err
	}
	err = datastore.DecodeBlob(oa, &n.Override)
	if err != nil {
		return err
	}
	datastore.ChkNilArray(n)
	return nil
}

func getSQL(nodeName string) (*Node, error) {
	node := new(Node)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from nodes n where n.name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from goiardi.nodes n where n.name = $1"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(nodeName)
	err = node.fillNodeFromSQL(row)

	if err != nil {
		return nil, err
	}
	return node, nil
}

func getMultiSQL(nodeNames []string) ([]*Node, error) {
	var sqlStmt string
	bind := make([]string, len(nodeNames))

	if config.Config.UseMySQL {
		for i := range nodeNames {
			bind[i] = "?"
		}
		sqlStmt = fmt.Sprintf("select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from nodes n where n.name in (%s)", strings.Join(bind, ", "))
	} else if config.Config.UsePostgreSQL {
		for i := range nodeNames {
			bind[i] = fmt.Sprintf("$%d", i+1)
		}
		sqlStmt = fmt.Sprintf("select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from goiardi.nodes n where n.name in (%s)", strings.Join(bind, ", "))
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	nameArgs := make([]interface{}, len(nodeNames))
	for i, v := range nodeNames {
		nameArgs[i] = v
	}
	rows, err := stmt.Query(nameArgs...)
	if err != nil {
		return nil, err
	}
	nodes := make([]*Node, 0, len(nodeNames))
	for rows.Next() {
		n := new(Node)
		err = n.fillNodeFromSQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		nodes = append(nodes, n)
	}

	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (n *Node) saveSQL() error {
	// prepare the complex structures for saving
	rlb, rlerr := datastore.EncodeBlob(&n.RunList)
	if rlerr != nil {
		return rlerr
	}
	aab, aaerr := datastore.EncodeBlob(&n.Automatic)
	if aaerr != nil {
		return aaerr
	}
	nab, naerr := datastore.EncodeBlob(&n.Normal)
	if naerr != nil {
		return naerr
	}
	dab, daerr := datastore.EncodeBlob(&n.Default)
	if daerr != nil {
		return daerr
	}
	oab, oaerr := datastore.EncodeBlob(&n.Override)
	if oaerr != nil {
		return oaerr
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	if config.Config.UseMySQL {
		err = n.saveMySQL(tx, rlb, aab, nab, dab, oab)
	} else if config.Config.UsePostgreSQL {
		err = n.savePostgreSQL(tx, rlb, aab, nab, dab, oab)
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (n *Node) deleteSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "DELETE FROM nodes WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "DELETE FROM goiardi.nodes WHERE name = $1"
	}

	_, err = tx.Exec(sqlStmt, n.Name)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting node %s had an error '%s', and then rolling back the transaction gave another error '%s'", n.Name, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return err
}

func deleteByAgeSQL(dur time.Duration) (int, error) {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return 0, err
	}
	from := time.Now().Add(-dur)

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "DELETE FROM node_statuses WHERE updated_at >= ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "DELETE FROM goiardi.node_statuses WHERE updated_at >= $1"
	}

	res, err := tx.Exec(sqlStmt, from)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting node statuses for the last %s had an error '%s', and then rolling back the transaction gave another error '%s'", from, err.Error(), terr.Error())
		}
		return 0, err
	}
	tx.Commit()
	rows, _ := res.RowsAffected()
	return int(rows), nil
}

func (ns *NodeStatus) updateNodeStatusSQL() error {
	if config.Config.UseMySQL {
		return ns.updateNodeStatusMySQL()
	} else if config.Config.UsePostgreSQL {
		return ns.updateNodeStatusPostgreSQL()
	}
	err := fmt.Errorf("reached an impossible db state")
	return err
}

func (ns *NodeStatus) importNodeStatus() error {
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "INSERT INTO node_statuses (node_id, status, updated_at) SELECT id, ?, ? FROM nodes WHERE name = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "INSERT INTO goiardi.node_statuses (node_id, status, updated_at) SELECT id, $1, $2 FROM goiardi.nodes WHERE name = $3"
	}
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStmt, ns.Status, ns.UpdatedAt, ns.Node.Name)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func getListSQL() []string {
	var nodeList []string
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT name FROM nodes"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT name FROM goiardi.nodes"
	}
	rows, err := datastore.Dbh.Query(sqlStmt)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return nodeList
	}
	for rows.Next() {
		var nodeName string
		err = rows.Scan(&nodeName)
		if err != nil {
			log.Fatal(err)
		}
		nodeList = append(nodeList, nodeName)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return nodeList
}

func getNodesInEnvSQL(envName string) ([]*Node, error) {
	var nodes []*Node
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM nodes n WHERE n.chef_environment = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM goiardi.nodes n WHERE n.chef_environment = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(envName)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nodes, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		n := new(Node)
		err = n.fillNodeFromSQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		nodes = append(nodes, n)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func allNodesSQL() []*Node {
	var nodes []*Node
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from nodes n"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from goiardi.nodes n"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nodes
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		no := new(Node)
		err = no.fillNodeFromSQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		nodes = append(nodes, no)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return nodes
}

func (n *Node) latestStatusSQL() (*NodeStatus, error) {
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT status, ns.updated_at FROM node_statuses ns JOIN nodes n on ns.node_id = n.id WHERE n.name = ? ORDER BY ns.id DESC LIMIT 1"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT status, updated_at FROM goiardi.node_latest_statuses WHERE name = $1"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	ns := &NodeStatus{Node: n}
	row := stmt.QueryRow(n.Name)
	if config.Config.UseMySQL {
		err = ns.fillNodeStatusFromMySQL(row)
	} else if config.Config.UsePostgreSQL {
		err = ns.fillNodeStatusFromPostgreSQL(row)
	}
	if err != nil {
		return nil, err
	}
	return ns, nil
}

func (n *Node) allStatusesSQL() ([]*NodeStatus, error) {
	var nodeStatuses []*NodeStatus
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT status, ns.updated_at FROM node_statuses ns JOIN nodes n on ns.node_id = n.id WHERE n.name = ? ORDER BY ns.id"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT status, ns.updated_at FROM goiardi.node_statuses ns JOIN goiardi.nodes n ON ns.node_id = n.id WHERE n.name = $1 ORDER BY ns.id"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(n.Name)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nodeStatuses, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		ns := &NodeStatus{Node: n}
		if config.Config.UseMySQL {
			err = ns.fillNodeStatusFromMySQL(rows)
		} else if config.Config.UsePostgreSQL {
			err = ns.fillNodeStatusFromPostgreSQL(rows)
		}
		if err != nil {
			return nil, err
		}
		nodeStatuses = append(nodeStatuses, ns)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return nodeStatuses, nil
}

func unseenNodesSQL() ([]*Node, error) {
	var nodes []*Node
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from nodes n join node_statuses ns on n.id = ns.node_id where is_down = 0 group by n.id having max(ns.updated_at) < date_sub(now(), interval 10 minute)"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from goiardi.node_latest_statuses n where n.is_down = false AND n.updated_at < now() - interval '10 minute'"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query()
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

func getNodesByStatusSQL(nodeNames []string, status string) ([]*Node, error) {
	if config.Config.UseMySQL {
		return getNodesByStatusMySQL(nodeNames, status)
	} else if config.Config.UsePostgreSQL {
		return getNodesByStatusPostgreSQL(nodeNames, status)
	}
	err := fmt.Errorf("impossible db state, man")
	return nil, err
}

func countSQL() (int64, error) {
	var c int64
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT COUNT(*) FROM nodes"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT COUNT(*) FROM goiardi.nodes"
	}
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return 0, err
	}
	err = stmt.QueryRow().Scan(&c)
	if err != nil {
		return 0, err
	}
	return c, nil
}
