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

/* Generic SQL functions for nodes */

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"log"
	"strings"
	"time"
)

func checkForNodeSQL(dbhandle datastore.Dbhandle, org *organization.Organization, name string) (bool, error) {
	_, err := datastore.CheckForOne(datastore.Dbh, "nodes", org.GetId(), name)
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

func getSQL(org *organization.Organization, nodeName string) (*Node, error) {
	node := new(Node)
	node.org = org

	sqlStmt := "SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM goiardi.nodes n WHERE n.organization_id = $1 AND n.name = $2"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(org.GetId(), nodeName)
	err = node.fillNodeFromSQL(row)

	if err != nil {
		return nil, err
	}
	return node, nil
}

func getMultiSQL(org *organization.Organization, nodeNames []string) ([]*Node, error) {
	bind := make([]string, len(nodeNames))

	for i := range nodeNames {
		bind[i] = fmt.Sprintf("$%d", i+2)
	}
	sqlStmt := fmt.Sprintf("SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM goiardi.nodes n WHERE n.organization_id = $1 AND n.name IN (%s)", strings.Join(bind, ", "))

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	nameArgs := make([]interface{}, len(nodeNames)+1)
	nameArgs[0] = org.GetId()
	for i, v := range nodeNames {
		nameArgs[i+1] = v
	}
	rows, err := stmt.Query(nameArgs...)
	if err != nil {
		return nil, err
	}
	nodes := make([]*Node, len(nodeNames))
	x := 0
	for rows.Next() {
		n := new(Node)
		n.org = org
		err = n.fillNodeFromSQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		nodes[x] = n
		x++
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

	if err = n.savePostgreSQL(tx, rlb, aab, nab, dab, oab); err != nil {
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
	sqlStmt := "DELETE FROM goiardi.nodes WHERE organization_id = $1 AND name = $2"

	_, err = tx.Exec(sqlStmt, n.org.GetId(), n.Name)
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

func deleteByAgeSQL(org *organization.Organization, dur time.Duration) (int, error) {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return 0, err
	}
	from := time.Now().Add(-dur)

	sqlStmt := "DELETE FROM goiardi.node_statuses ns JOIN goiardi.nodes ON ns.node_id = n.id WHERE n.organization_id = $1 AND ns.updated_at >= $2"

	res, err := tx.Exec(sqlStmt, org.GetId(), from)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting node statuses in org '%s' for the last %s had an error '%s', and then rolling back the transaction gave another error '%s'", org.Name, from, err.Error(), terr.Error())
		}
		return 0, err
	}
	tx.Commit()
	rows, _ := res.RowsAffected()
	return int(rows), nil
}

func (ns *NodeStatus) updateNodeStatusSQL() error {
	return ns.updateNodeStatusPostgreSQL()
}

func (ns *NodeStatus) importNodeStatus() error {
	sqlStmt := "INSERT INTO goiardi.node_statuses (node_id, status, updated_at) SELECT id, $1, $2 FROM goiardi.nodes WHERE organization_id = $3 AND name = $4"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(sqlStmt, ns.Status, ns.UpdatedAt, ns.Node.org.GetId(), ns.Node.Name)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func getListSQL(org *organization.Organization) []string {
	var nodeList []string

	sqlStmt := "SELECT name FROM goiardi.nodes WHERE organization_id = $1"

	rows, err := datastore.Dbh.Query(sqlStmt, org.GetId())
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

func getNodesInEnvSQL(org *organization.Organization, envName string) ([]*Node, error) {
	var nodes []*Node
	sqlStmt := "SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM goiardi.nodes n WHERE n.organization_id = $1 AND n.chef_environment = $2"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(org.GetId(), envName)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nodes, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		n := new(Node)
		n.org = org
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

func allNodesSQL(org *organization.Organization) []*Node {
	var nodes []*Node
	sqlStmt := "SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM goiardi.nodes n WHERE n.organization_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(org.GetId())
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nodes
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		no := new(Node)
		no.org = org
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
	sqlStmt := "SELECT status, updated_at FROM goiardi.node_latest_statuses WHERE organization_id = $1 AND name = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	ns := &NodeStatus{Node: n}
	row := stmt.QueryRow(n.org.GetId(), n.Name)

	if err = ns.fillNodeStatusFromPostgreSQL(row); err != nil {
		return nil, err
	}
	return ns, nil
}

func (n *Node) allStatusesSQL() ([]*NodeStatus, error) {
	var nodeStatuses []*NodeStatus
	sqlStmt := "SELECT status, ns.updated_at FROM goiardi.node_statuses ns JOIN goiardi.nodes n ON ns.node_id = n.id WHERE n.organization_id = $1 AND n.name = $2 ORDER BY ns.id"
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(n.org.GetId(), n.Name)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return nodeStatuses, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		ns := &NodeStatus{Node: n}
		if err = ns.fillNodeStatusFromPostgreSQL(rows); err != nil {
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
	sqlStmt := "SELECT n.name, chef_environment, n.run_list, n.automatic_attr, n.normal_attr, n.default_attr, n.override_attr FROM goiardi.node_latest_statuses n WHERE AND n.is_down = false AND n.updated_at < NOW() - INTERVAL '10 minute'"

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

func getNodesByStatusSQL(org *organization.Organization, nodeNames []string, status string) ([]*Node, error) {
	return getNodesByStatusPostgreSQL(org, nodeNames, status)
}

func countSQL() (int64, error) {
	var c int64

	sqlStmt := "SELECT COUNT(*) FROM goiardi.nodes"

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

func orgCountSQL(org *organization.Organization) (int64, error) {
	var c int64

	sqlStmt := "SELECT COUNT(*) FROM goiardi.nodes WHERE organization_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return 0, err
	}
	err = stmt.QueryRow(org.GetId()).Scan(&c)
	if err != nil {
		return 0, err
	}
	return c, nil
}
