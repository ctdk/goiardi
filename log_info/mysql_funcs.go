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
	"github.com/ctdk/goiardi/actor"
	"database/sql"
)

func (le *LogInfo)writeEventMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	actor_id, err := data_store.CheckForOne(tx, le.ActorType, le.Actor.GetName())
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err := tx.Exec("INSERT INTO log_infos (actor_id, actor_type, time, action, object_type, object_name, extended_info) VALUES (?, ?, ?, ?, ?, ?, ?)", actor_id, le.ActorType, le.Time, le.Action, le.ObjectType, le.ObjectName, le.ExtendedInfo)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func getLogEventMySQL(id int) (*LogInfo, error) {
	
}

func fillLogEventFromMySQL(row data_store.ResRow) error {

}

func (le *LogInfo)deleteMySQL() error {

}

func purgeMySQL(id int) (int, error) {

}

func getLogInfoListMySQL(limits ...int) []*LogInfo {

}
