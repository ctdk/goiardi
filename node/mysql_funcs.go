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
	"github.com/ctdk/goiardi/data_store"
	"fmt"
	"log"
	"database/sql"
)

func checkForNodeMySQL(dbhandle data_store.Dbhandle, name string) (bool, error) {
	_, err := data_store.CheckForOne(data_store.Dbh, "nodes", name)
	if err == nil {
		return true, nil
	} else {
		if err != sql.ErrNoRows {
			return false, err
		} else {
			return false, nil
		}
	}
}

// Fill in a node from a row returned from the SQL server. Useful for the case
// down the road where an array of objects is needed, but building it with
// a call to GetList(), then repeated calls to Get() sucks with a real db even
// if it's marginally acceptable in in-memory mode.
//
// NB: This does require the query to look like the one in Get().
func (n *Node) fillNodeFromSQL(row data_store.ResRow) error {
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
	n.JsonClass = "Chef::Node"
	err = data_store.DecodeBlob(rl, &n.RunList)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(aa, &n.Automatic)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(na, &n.Normal)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(da, &n.Default)
	if err != nil {
		return err
	}
	err = data_store.DecodeBlob(oa, &n.Override)
	if err != nil {
		return err
	}
	data_store.ChkNilArray(n)
	return nil
}

func getMySQL(node_name string) (*Node, error){
	node := new(Node)
	stmt, err := data_store.Dbh.Prepare("select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from nodes n where n.name = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(node_name)
	err = node.fillNodeFromSQL(row)

	if err != nil {
		return nil, err
	}
	return node, nil
}

func (n *Node) saveMySQL() error {
	// prepare the complex structures for saving
	rlb, rlerr := data_store.EncodeBlob(&n.RunList)
	if rlerr != nil {
		return rlerr
	}
	aab, aaerr := data_store.EncodeBlob(&n.Automatic)
	if aaerr != nil {
		return aaerr
	}
	nab, naerr := data_store.EncodeBlob(&n.Normal)
	if naerr != nil {
		return naerr
	}
	dab, daerr := data_store.EncodeBlob(&n.Default)
	if daerr != nil {
		return daerr
	}
	oab, oaerr := data_store.EncodeBlob(&n.Override)
	if oaerr != nil {
		return oaerr
	}

	tx, err := data_store.Dbh.Begin()
	var node_id int32
	if err != nil {
		return err
	}
	// This does not use the INSERT ... ON DUPLICATE KEY UPDATE
	// syntax to keep the MySQL code & the future Postgres code
	// closer together.
	node_id, err = data_store.CheckForOne(tx, "nodes", n.Name)
	if err == nil {
		// probably want binlog_format set to MIXED or ROW for 
		// this query
		_, err := tx.Exec("UPDATE nodes n SET chef_environment = ?, n.run_list = ?, n.automatic_attr = ?, n.normal_attr = ?, n.default_attr = ?, n.override_attr = ?, n.updated_at = NOW() WHERE n.id = ?", n.ChefEnvironment, rlb, aab, nab, dab, oab, node_id)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		if err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec("INSERT INTO nodes (name, chef_environment, run_list, automatic_attr, normal_attr, default_attr, override_attr, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())", n.Name, n.ChefEnvironment, rlb, aab, nab, dab, oab)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (n *Node) deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM nodes WHERE name = ?", n.Name)
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

func getListMySQL() []string {
	node_list := make([]string, 0)
	rows, err := data_store.Dbh.Query("SELECT name FROM nodes")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return node_list
	}
	for rows.Next() {
		var node_name string
		err = rows.Scan(&node_name)
		if err != nil {
			log.Fatal(err)
		}
		node_list = append(node_list, node_name)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return node_list
}

func getNodesInEnvMySQL(env_name string) ([]*Node, error) {
	nodes := make([]*Node, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM nodes n WHERE n.chef_environment = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(env_name)
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
	nodes := make([]*Node, 0)
	stmt, err := data_store.Dbh.Prepare("select n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr from nodes n")
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
