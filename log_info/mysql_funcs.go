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

package log_info

import (
	"github.com/ctdk/goiardi/data_store"
	"database/sql"
	"time"
	"log"
	"fmt"
)

func (le *LogInfo)writeEventMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	type_table := fmt.Sprintf("%ss", le.ActorType)
	actor_id, err := data_store.CheckForOne(tx, type_table, le.Actor.GetName())
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec("INSERT INTO log_infos (actor_id, actor_type, actor_info, time, action, object_type, object_name, extended_info) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", actor_id, le.ActorType, le.ActorInfo, le.Time, le.Action, le.ObjectType, le.ObjectName, le.ExtendedInfo)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func getLogEventMySQL(id int) (*LogInfo, error) {
	le := new(LogInfo)
	stmt, err := data_store.Dbh.Prepare("SELECT id, actor_type, actor_info, time, action, object_type, object_name, extended_info FROM log_infos WHERE id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(id)
	err = le.fillLogEventFromMySQL(row)
	if err != nil {
		return nil, err
	}
	// conveniently, le.Actor does not seem to need to be populated after
	// it's been saved.
	return le, nil
}

func (le *LogInfo)fillLogEventFromMySQL(row data_store.ResRow) error {
	var tb []byte
	err := row.Scan(&le.Id, &le.ActorType, &le.ActorInfo, &tb, &le.Action, &le.ObjectType, &le.ObjectName, &le.ExtendedInfo)
	if err != nil {
		return err
	}
	le.Time, err = time.Parse(data_store.MySQLTimeFormat, string(tb))
	if err != nil {
		return err
	}
	return nil
}

func (le *LogInfo)deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec("DELETE FROM log_infos WHERE id = ?", le.Id)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func purgeMySQL(id int) (int64, error) {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return 0, err
	}
	res, err := tx.Exec("DELETE FROM log_infos WHERE id <= ?", id)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	rows_affected, _ := res.RowsAffected()
	tx.Commit()
	return rows_affected, nil
}

func getLogInfoListMySQL(limits ...int) []*LogInfo {
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
	logged_events := make([]*LogInfo, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT id, actor_type, actor_info, time, action, object_type, object_name, extended_info FROM log_infos LIMIT ?, ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, qerr := stmt.Query(offset, limit)
	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return logged_events
		}
		log.Fatal(qerr)
	}
	for rows.Next() {
		le := new(LogInfo)
		err = le.fillLogEventFromMySQL(rows)
		if err != nil {
			log.Fatal(err)
		}
		logged_events = append(logged_events, le)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return logged_events
}
