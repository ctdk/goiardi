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

package report

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/util"
	"database/sql"
	"fmt"
	"log"
)

func checkForReportMySQL(dbhandle data_store.Dbhandle, runId string) (bool, error) {
	var f int
	stmt, err := dbhandle.Prepare("SELECT count(*) AS c FROM reports WHERE run_id = ?")
	if err != nil {
		return false, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(runId).Scan(&f)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		} else {
			return false, err
		}
	}
	if f > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (r *Report)fillReportFromMySQL(row data_store.ResRow) error{

}

func getReportMySQL(runId string) (*Report, error) {

}

func (r *Report)saveMySQL() err {

}

func (r *Report)deleteMySQL() error {

}

func getListMySQL() []string {

}

func getReportListMySQL() []*Report {

}

func getNodeListMySQL(nodeName string) []*Report {

}
