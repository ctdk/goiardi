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

package loginfo

/* Generic SQL functions for logging events */

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"io/ioutil"
	"regexp"
	"time"
)

func (le *LogInfo) writeEventSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	typeTable := fmt.Sprintf("%ss", le.ActorType)
	actorID, err := datastore.CheckForOne(tx, typeTable, le.OrgId(), le.Actor.GetName())
	if err != nil {
		tx.Rollback()
		return err
	}
	err = le.actualWriteEventSQL(tx, actorID)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (le *LogInfo) importEventSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	typeTable := fmt.Sprintf("%ss", le.ActorType)

	aiBuf := bytes.NewBuffer([]byte(le.ActorInfo))
	aiRC := ioutil.NopCloser(aiBuf)
	doer := make(map[string]interface{})

	dec := json.NewDecoder(aiRC)
	dec.UseNumber()
	if err = dec.Decode(&doer); err != nil {
		tx.Rollback()
		return err
	}

	actorID, err := datastore.CheckForOne(tx, typeTable, le.OrgId(), doer["name"].(string))
	if err != nil {
		actorID = -1
	}
	err = le.actualWriteEventSQL(tx, actorID)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

// This has been broken out to a separate function to simplify importing data
// from json export dumps.
func (le *LogInfo) actualWriteEventSQL(tx datastore.Dbhandle, actorID int32) error {
	return le.actualWriteEventPostgreSQL(tx, actorID)
}

func getLogEventSQL(id int, orgId int64) (*LogInfo, error) {
	le := new(LogInfo)

	sqlStmt := "SELECT id, actor_type, actor_info, time, action, object_type, object_name, extended_info FROM goiardi.log_infos WHERE organization_id = $1 AND id = $2"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	if orgId != 0 {
		org, oerr := orgloader.OrgByIdSQL(orgId)
		if oerr != nil {
			return nil, oerr
		}
		le.org = org
	}

	row := stmt.QueryRow(orgId, id)
	if err = le.fillLogEventFromPostgreSQL(row); err != nil {
		return nil, err
	}

	// conveniently, le.Actor does not seem to need to be populated after
	// it's been saved.
	return le, nil
}

func checkLogEventSQL(id int, orgId int64) (bool, error) {
	var found bool

	sqlStmt := "SELECT id FROM goiardi.log_infos WHERE organization_id = $1 AND id = $2"

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var c int
	err = stmt.QueryRow(orgId, id).Scan(&c)
	// should be hard at best to get ErrNoRows in this situation
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if c != 0 {
		found = true
	}
	return found, nil
}

func (le *LogInfo) deleteSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	sqlStmt := "DELETE FROM goiardi.log_infos WHERE organization_id = $1 AND id = $2"

	_, err = tx.Exec(sqlStmt, le.OrgId(), le.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func purgeSQL(id int, orgId int64) (int64, error) {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return 0, err
	}

	sqlStmt := "DELETE FROM goiardi.log_infos WHERE organization_id = $1 AND id <= $2"

	res, err := tx.Exec(sqlStmt, orgId, id)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	rowsAffected, _ := res.RowsAffected()
	tx.Commit()
	return rowsAffected, nil
}

func getLogInfoListSQL(orgId int64, searchParams map[string]string, from, until time.Time, limits ...int) ([]*LogInfo, error) {
	var offset int
	var limit int64 = (1 << 63) - 1
	if len(limits) > 0 {
		offset = limits[0]
		if len(limits) > 1 {
			limit = int64(limits[1])
		}
	} else {
		offset = 0
	}
	var loggedEvents []*LogInfo

	sqlArgs := []interface{}{orgId, from, until}

	sqlStmt := "SELECT li.id, actor_type, actor_info, time, action, object_type, object_name, extended_info FROM goiardi.log_infos li JOIN goiardi.users u ON li.actor_id = u.id WHERE organization_id = ? AND time >= ? AND time <= ?"
	if action, ok := searchParams["action"]; ok {
		sqlStmt = sqlStmt + " AND action = ?"
		sqlArgs = append(sqlArgs, action)
	}
	if objectType, ok := searchParams["object_type"]; ok {
		sqlStmt = sqlStmt + " AND object_type = ?"
		sqlArgs = append(sqlArgs, objectType)
	}
	if objectName, ok := searchParams["object_name"]; ok {
		sqlStmt = sqlStmt + " AND object_name = ?"
		sqlArgs = append(sqlArgs, objectName)
	}
	if doer, ok := searchParams["doer"]; ok {
		sqlStmt = sqlStmt + " AND u.name = ?"
		sqlArgs = append(sqlArgs, doer)
	} else {
		re := regexp.MustCompile("JOIN goiardi.users u ON li.actor_id = u.id")
		sqlStmt = re.ReplaceAllString(sqlStmt, "")
	}
	sqlStmt = sqlStmt + " ORDER BY id DESC OFFSET ? LIMIT ?"
	re := regexp.MustCompile("\\?")
	u := 1
	rfunc := func([]byte) []byte {
		r := []byte(fmt.Sprintf("$%d", u))
		u++
		return r
	}
	sqlStmt = string(re.ReplaceAllFunc([]byte(sqlStmt), rfunc))

	sqlArgs = append(sqlArgs, offset)
	sqlArgs = append(sqlArgs, limit)

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(sqlArgs...)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return loggedEvents, nil
		}
		return nil, qerr
	}

	var org *organization.Organization
	if orgId != 0 {
		var oerr error
		org, oerr = orgloader.OrgByIdSQL(orgId)
		if oerr != nil {
			return nil, oerr
		}
	}

	for rows.Next() {
		le := new(LogInfo)
		le.org = org
		if err = le.fillLogEventFromPostgreSQL(rows); err != nil {
			return nil, err
		}
		loggedEvents = append(loggedEvents, le)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return loggedEvents, nil
}
