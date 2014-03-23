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

// General functions for goiardi database connections, if running in that mode.
// Database engine specific functions are in their respective source files.
package data_store

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"strings"
	"fmt"
)

var Dbh *sql.DB

// Connect to a database with the database name and a map of connection options.
func ConnectDB(dbEngine string, params interface{}) (*sql.DB, error) {
	switch strings.ToLower(dbEngine) {
		case "mysql":
			connectStr, cerr := formatMysqlConStr(params)
			if cerr != nil {
				return nil, cerr
			}
			db, err := sql.Open(strings.ToLower(dbEngine), connectStr)
			if err != nil {
				return nil, err
			}
			if err = db.Ping(); err != nil {
				return nil, err
			}
			return db, nil
		default:
			err := fmt.Errorf("cannot connect to database: unsupported database type %s", dbEngine)
			return nil, err
	}
}
