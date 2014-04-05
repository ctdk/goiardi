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

package data_bag

import (
	"github.com/ctdk/goiardi/data_store"
	"database/sql"
	"fmt"
)

// Functions for finding, saving, etc. data bags with a MySQL database.

func checkForDataBagMySQL(name string) (bool, error) {
	_, err := data_store.CheckForOne(data_store.Dbh, "data_bags", name)
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

func getDataBagMySQL(name string) (*DataBag, error) {

}

func (db *DataBag) getDBItemMySQL(db_item_name string) (*DataBagItem, error) {

}

func (db *DataBag) allDBItemsMySQL()(map[string]*DataBagItem, error) {

}

func (db *DataBag) numDBItemsMySQL() int {

}

func (db *DataBag) listDBItemsMySQL() []string {

}
