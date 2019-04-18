/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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

/* Generic SQL funcs for reports */

import (
	"database/sql"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"log"
	"time"
)

func checkForReportSQL(dbhandle datastore.Dbhandle, runID string) (bool, error) {
	var f int
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT count(*) AS c FROM reports WHERE run_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT count(*) AS c FROM goiardi.reports WHERE run_id = $1"
	}
	stmt, err := dbhandle.Prepare(sqlStmt)
	if err != nil {
		return false, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(runID).Scan(&f)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if f > 0 {
		return true, nil
	}
	return false, nil
}

func (r *Report) fillReportFromSQL(row datastore.ResRow) error {
	if config.Config.UseMySQL {
		return r.fillReportFromMySQL(row)
	} else if config.Config.UsePostgreSQL {
		return r.fillReportFromPostgreSQL(row)
	}

	return nil
}

func getReportSQL(runID string) (*Report, error) {
	r := new(Report)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE run_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE run_id = $1"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(runID)
	err = r.fillReportFromSQL(row)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Report) deleteSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "DELETE FROM reports WHERE run_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "DELETE FROM goiardi.reports WHERE run_id = $1"
	}

	_, err = tx.Exec(sqlStmt, r.RunID)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting report %s had an error '%s', and then rolling back the transaction gave another error '%s'", r.RunID, err.Error(), terr.Error())
		}
		return err
	}
	tx.Commit()
	return nil
}

func deleteByAgeSQL(dur time.Duration) (int, error) {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return 0, err
	}
	from := time.Now().Add(-dur)

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "DELETE FROM reports WHERE end_time >= ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "DELETE FROM goiardi.reports WHERE end_time >= $1"
	}

	res, err := tx.Exec(sqlStmt, from)
	if err != nil {
		terr := tx.Rollback()
		if terr != nil {
			err = fmt.Errorf("deleting reports for the last %s had an error '%s', and then rolling back the transaction gave another error '%s'", from, err.Error(), terr.Error())
		}
		return 0, err
	}
	tx.Commit()
	rows, _ := res.RowsAffected()
	return int(rows), nil
}

func getListSQL() []string {
	var reportList []string

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id FROM reports"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id FROM goiardi.reports"
	}

	rows, err := datastore.Dbh.Query(sqlStmt)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatal(err)
		}
		rows.Close()
		return reportList
	}
	for rows.Next() {
		var runID string
		err = rows.Scan(&runID)
		if err != nil {
			log.Fatal(err)
		}
		reportList = append(reportList, runID)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return reportList
}

func getReportListSQL(from, until time.Time, retrows int, status string) ([]*Report, error) {
	var reports []*Report
	var sqlStmt string

	if status == "" {
		if config.Config.UseMySQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE start_time >= ? AND start_time <= ? LIMIT ?"
		} else if config.Config.UsePostgreSQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE start_time >= $1 AND start_time <= $2 LIMIT $3"
		}
	} else {
		if config.Config.UseMySQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE start_time >= ? AND start_time <= ? AND status = ? LIMIT ?"
		} else if config.Config.UsePostgreSQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE start_time >= $1 AND start_time <= $2 AND status = $3 LIMIT $4"
		}
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var rows *sql.Rows
	var rerr error

	if status == "" {
		rows, rerr = stmt.Query(from, until, retrows)
	} else {
		rows, rerr = stmt.Query(from, until, status, retrows)
	}
	if rerr != nil {
		if rerr == sql.ErrNoRows {
			return reports, nil
		}
		return nil, rerr
	}
	for rows.Next() {
		r := new(Report)
		err = r.fillReportFromSQL(rows)
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

func getNodeListSQL(nodeName string, from, until time.Time, retrows int, status string) ([]*Report, error) {
	var reports []*Report

	var sqlStmt string
	if status == "" {
		if config.Config.UseMySQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE node_name = ? AND start_time >= ? AND start_time <= ? LIMIT ?"
		} else if config.Config.UsePostgreSQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE node_name = $1 AND start_time >= $2 AND start_time <= $3 LIMIT $4"
		}
	} else {
		if config.Config.UseMySQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE node_name = ? AND start_time >= ? AND start_time <= ? AND status = ? LIMIT ?"
		} else if config.Config.UsePostgreSQL {
			sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE node_name = $1 AND start_time >= $2 AND start_time <= $3 AND status = $4 LIMIT $5"
		}
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var rows *sql.Rows
	var rerr error
	if status == "" {
		rows, rerr = stmt.Query(nodeName, from, until, retrows)
	} else {
		rows, rerr = stmt.Query(nodeName, from, until, status, retrows)
	}
	if rerr != nil {
		if rerr == sql.ErrNoRows {
			return reports, nil
		}
		return nil, rerr
	}
	for rows.Next() {
		r := new(Report)
		err = r.fillReportFromSQL(rows)
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

func getReportsSQL() []*Report {
	var reports []*Report

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports"
	}

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	rows, rerr := stmt.Query()
	if rerr != nil {
		if rerr == sql.ErrNoRows {
			return reports
		}
		log.Fatal(rerr)
	}
	for rows.Next() {
		r := new(Report)
		err = r.fillReportFromSQL(rows)
		if err != nil {
			rows.Close()
			log.Fatal(err)
		}
		reports = append(reports, r)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return reports
}
