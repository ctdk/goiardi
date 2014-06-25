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

/* Generic SQL funcs for reports */

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/config"
	"database/sql"
	"fmt"
	"log"
	"time"
)

func checkForReportSQL(dbhandle data_store.Dbhandle, runId string) (bool, error) {
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

func (r *Report)fillReportFromSQL(row data_store.ResRow) error{
	if config.Config.UseMySQL {
		return r.fillReportFromMySQL(row)
	} else if config.Config.UsePostgreSQL {
		return r.fillReportFromPostgreSQL(row)
	}

	return nil
}

func getReportSQL(runId string) (*Report, error) {
	r := new(Report)
	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE run_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE run_id = $1"
	}

	stmt, err := data_store.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(runId)
	err = r.fillReportFromSQL(row)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Report)deleteSQL() error {
	tx, err := data_store.Dbh.Begin()
	if err != nil {
		return nil
	}

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "DELETE FROM reports WHERE run_id = ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "DELETE FROM goiardi.reports WHERE run_id = $1"
	}

	_, err = tx.Exec(sqlStmt, r.RunId)
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

func getListSQL() []string {
	reportList := make([]string, 0)

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id FROM reports"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id FROM goiardi.reports"
	}

	rows, err := data_store.Dbh.Query(sqlStmt)
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

func getReportListSQL(from, until time.Time, retrows int) ([]*Report, error) {
	reports := make([]*Report, 0)
	var sqlStmt string

	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE start_time >= ? AND start_time <= ? LIMIT ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE start_time >= $1 AND start_time <= $2 LIMIT $3"
	}

	stmt, err := data_store.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, rerr := stmt.Query(from, until, retrows)
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

func getNodeListSQL(nodeName string, from, until time.Time, retrows int) ([]*Report, error) {
	reports := make([]*Report, 0)

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports WHERE node_name = ? AND start_time >= ? AND start_time <= ? LIMIT ?"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports WHERE node_name = $1 AND start_time >= $2 AND start_time <= $3 LIMIT $4"
	}

	stmt, err := data_store.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, rerr := stmt.Query(nodeName, from, until, retrows)
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
	reports := make([]*Report, 0)

	var sqlStmt string
	if config.Config.UseMySQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM reports"
	} else if config.Config.UsePostgreSQL {
		sqlStmt = "SELECT run_id, start_time, end_time, total_res_count, status, run_list, resources, data, node_name FROM goiardi.reports"
	}

	stmt, err := data_store.Dbh.Prepare(sqlStmt)
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
