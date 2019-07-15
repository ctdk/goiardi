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

package group

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"github.com/lib/pq"
	"github.com/tideland/golib/logger"
	"net/http"
)

// anything specific enough to postgres that weird conditionals inside a common
// function or method would be unwieldy.

func (g *Group) savePostgreSQL(userIds []int64, clientIds []int64, groupIds []int64) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	// hrm.
	logger.Debugf("group org id: %d -- user ids: '%s' client ids: '%v' group ids: '%v'", g.org.GetId(), userIds, clientIds, groupIds)

	_, err = tx.Exec("SELECT goiardi.merge_groups($1, $2, $3, $4, $5)", g.Name, g.org.GetId(), pq.Int64Array(userIds), pq.Int64Array(clientIds), pq.Int64Array(groupIds))
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}
