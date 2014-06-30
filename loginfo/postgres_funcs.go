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

package loginfo

/* PostgreSQL specific functions for loginfo */

import (
	"github.com/ctdk/goiardi/datastore"
)

func (le *LogInfo) fillLogEventFromPostgreSQL(row datastore.ResRow) error {
	err := row.Scan(&le.ID, &le.ActorType, &le.ActorInfo, &le.Time, &le.Action, &le.ObjectType, &le.ObjectName, &le.ExtendedInfo)
	if err != nil {
		return err
	}
	return nil
}

func (le *LogInfo) actualWriteEventPostgreSQL(tx datastore.Dbhandle, actorID int32) error {
	var err error
	if le.ID == 0 {
		sqlStmt := "INSERT INTO goiardi.log_infos (actor_id, actor_type, actor_info, time, action, object_type, object_name, extended_info) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"
		_, err = tx.Exec(sqlStmt, actorID, le.ActorType, le.ActorInfo, le.Time, le.Action, le.ObjectType, le.ObjectName, le.ExtendedInfo)
	} else {
		sqlStmt := "INSERT INTO goiardi.log_infos (id, actor_id, actor_type, actor_info, time, action, object_type, object_name, extended_info) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
		_, err = tx.Exec(sqlStmt, le.ID, actorID, le.ActorType, le.ActorInfo, le.Time, le.Action, le.ObjectType, le.ObjectName, le.ExtendedInfo)
	}
	return err
}
