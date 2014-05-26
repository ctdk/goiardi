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
	"database/sql"
	"github.com/go-sql-driver/mysql"
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
	var res, dat []byte
	var st, et mysql.NullTime
	err := row.Scan(&r.RunId, &st, &et, &r.TotalResCount, &r.Status, &r.RunList, &res, &dat, &r.NodeName)
	if err != nil {
		return err
	}
	if err = data_store.DecodeBlob(res, &r.Resources); err != nil {
		return err
	}
	if err = data_store.DecodeBlob(dat, &r.Data); err != nil {
		return err
	}
	if st.Valid {
		r.StartTime = st.Time
	} 
	if et.Valid {
		r.EndTime = et.Time
	}

	return nil
}

func getReportMySQL(runId string) (*Report, error) {
	r := new(Report)
	stmt, err := data_store.Dbh.Prepare("SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE run_id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(runId)
	err = r.fillReportFromMySQL(row)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Report)saveMySQL() error {
	res, reserr := data_store.EncodeBlob(&r.Resources)
	if reserr != nil {
		return reserr
	}
	dat, daterr := data_store.EncodeBlob(&r.Data)
	if daterr != nil {
		return daterr
	}
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return err
	}
	// Up to this point I was going the INSERT or UPDATE without using
	// MySQL specific syntax, to keep MySQL and any future Postgres
	// SQL more similar, but now I'm thinking that this should try and
	// leverage more of each database's capabilities. Thus, here we shall
	// do the very MySQL-specific INSERT ... ON DUPLICATE KEY UPDATE
	// syntax.
	_, err = tx.Exec("INSERT INTO reports (run_id, node_name, start_time, end_time, total_res_count, status, run_list, resources, data, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE start_time = ?, end_time = ?, total_res_count = ?, status = ?, run_list = ?, resources = ?, data = ?, updated_at = NOW()", r.RunId, r.NodeName, r.StartTime, r.EndTime, r.TotalResCount, r.Status, r.RunList, res, dat, r.StartTime, r.EndTime, r.TotalResCount, r.Status, r.RunList, res, dat)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (r *Report)deleteMySQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return nil
	}
	_, err = tx.Exec("DELETE FROM reports WHERE run_id = ?", r.RunId)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting report %s had an error '%s', and then rolling back the transaction gave another error '%s'", r.RunId, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func getListMySQL() []string {
	reportList := make([]string, 0)
	rows, err := data_store.Dbh.Query("SELECT run_id FROM reports")
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return reportList
	}
	for rows.Next() {
		var runId string
		err = rows.Scan(&runId)
		if err != nil {
			log.Fatal(err)
		}
		reportList = append(reportList, runId)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return reportList
}

func getReportListMySQL() ([]*Report, error) {
	reports := make([]*Report, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, rerr := stmt.Query()
	if rerr != nil {
		if rerr == sql.ErrNoRows {
			return reports, nil
		}
		return nil, rerr
	}
	for rows.Next() {
		r := new(Report)
		err = r.fillReportFromMySQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		reports = append(reports, r)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return reports, nil
}

func getNodeListMySQL(nodeName string) ([]*Report, error) {
	reports := make([]*Report, 0)
	stmt, err := data_store.Dbh.Prepare("SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE node_name = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, rerr := stmt.Query(nodeName)
	if rerr != nil {
		if rerr == sql.ErrNoRows {
			return reports, nil
		}
		return nil, rerr
	}
	for rows.Next() {
		r := new(Report)
		err = r.fillReportFromMySQL(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		reports = append(reports, r)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return reports, nil
}
