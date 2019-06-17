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
)

// anything specific enough to postgres that weird conditionals inside a common
// function or method would be unwieldy.

func (g *Group) savePostgreSQL(user_ids []int64, client_ids []int64, group_ids []int64) error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.Errorf(err.Error())
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.merge_groups($1, $2, $3, $4, $5)", g.Name, g.Org.GetId(), user_ids, client_ids, group_ids)
	if err != nil {
		tx.Rollback()
		gerr := util.Errorf(err.Error())
		if strings.HasPrefix(err.Error(), strings.Contains(err.Error(), "already exists, cannot rename")) {
			gerr.SetStatus(http.StatusConflict)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return gerr
	}
	g.Name = newName
	tx.Commit()
	return nil
}
