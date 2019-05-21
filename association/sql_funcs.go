/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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

package association

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"log"
	"time"
)

func checkForAssociationSQL(dbhandle datastore.Dbhandle, user *user.User, org *organization.Organization) (bool, error) {
	var z int
	var sqlStmt string
	if config.Config.UseMySQL {
		// come back if we decide to actually keep mysql still - it's
		// iffy
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT count(*) AS c FROM goiardi.associations WHERE user_id = $1 AND organization_id = $2"
	}

	stmt, err := dbhandle.Prepare(sqlStmt)
	if err != nil {
		return false, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(user.GetId(), org.GetId()).Scan(&z)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if z > 0 {
		return true, nil
	}
	return false, nil
}
