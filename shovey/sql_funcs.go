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

package shovey

import (
	"database/sql"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

func checkForShoveySQL(org *organization.Organization, runID string) (bool, error) {
	var f int
	sqlStmt := "SELECT count(*) AS c FROM goiardi.shoveys WHERE organization_id = $1 AND run_id = $2"
	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return false, err
	}
	defer stmt.Close()
	err = stmt.QueryRow(org.GetId(), runID).Scan(&f)
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

func (s *Shovey) fillShoveyFromSQL(row datastore.ResRow) error {
	return s.fillShoveyFromPostgreSQL(row)
}

func (sr *ShoveyRun) fillShoveyRunFromSQL(row datastore.ResRow) error {
	return sr.fillShoveyRunFromPostgreSQL(row)
}

func (srs *ShoveyRunStream) fillShoveyRunStreamFromSQL(row datastore.ResRow) error {
	return srs.fillShoveyRunStreamFromPostgreSQL(row)
}

func getShoveySQL(org *organization.Organization, runID string) (*Shovey, util.Gerror) {
	s := new(Shovey)
	s.org = org
	sqlStatement := "SELECT run_id, ARRAY(SELECT node_name FROM goiardi.shovey_runs WHERE shovey_uuid = $1), command, created_at, updated_at, status, timeout, quorum FROM goiardi.shoveys WHERE organization_id = $2 AND run_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	defer stmt.Close()
	row := stmt.QueryRow(runID, org.GetId())
	err = s.fillShoveyFromSQL(row)
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return nil, gerr
	}

	return s, nil
}

func (s *Shovey) getShoveyRunSQL(nodeName string) (*ShoveyRun, util.Gerror) {
	sr := new(ShoveyRun)
	sr.org = s.org 		// may not really be necessary

	sqlStatement := "SELECT id, shovey_uuid, node_name, status, ack_time, end_time, error, exit_status FROM goiardi.shovey_runs WHERE shovey_uuid = $1 and node_name = $2"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	defer stmt.Close()
	row := stmt.QueryRow(s.RunID, nodeName)
	err = sr.fillShoveyRunFromSQL(row)
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return nil, gerr
	}

	return sr, nil
}

func (s *Shovey) getShoveyNodeRunsSQL() ([]*ShoveyRun, util.Gerror) {
	var shoveyRuns []*ShoveyRun

	sqlStatement := "SELECT id, shovey_uuid, node_name, status, ack_time, end_time, error, exit_status FROM goiardi.shovey_runs WHERE shovey_uuid = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	defer stmt.Close()
	rows, err := stmt.Query(s.RunID)
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		rows.Close()
		return nil, gerr
	}
	for rows.Next() {
		sr := new(ShoveyRun)
		sr.org = s.org
		err = sr.fillShoveyRunFromSQL(rows)
		if err != nil {
			gerr := util.CastErr(err)
			if err == sql.ErrNoRows {
				gerr.SetStatus(http.StatusNotFound)
			} else {
				gerr.SetStatus(http.StatusInternalServerError)
			}
			return nil, gerr
		}
		shoveyRuns = append(shoveyRuns, sr)
	}

	return shoveyRuns, nil
}

func (s *Shovey) saveSQL() util.Gerror {
	return s.savePostgreSQL()
}

func (sr *ShoveyRun) saveSQL() util.Gerror {
	return sr.savePostgreSQL()
}

func (s *Shovey) cancelRunsSQL() util.Gerror {
	sqlStatement := "UPDATE goiardi.shovey_runs SET status = 'cancelled', end_time = NOW() WHERE shovey_uuid = $1 AND status NOT IN ('invalid', 'succeeded', 'failed', 'down', 'nacked')"

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	_, err = tx.Exec(sqlStatement, s.RunID)
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return gerr
	}
	tx.Commit()
	return nil
}

func (s *Shovey) checkCompletedSQL() util.Gerror {
	var c int
	sqlStatement := "SELECT count(id) FROM goiardi.shovey_runs WHERE shovey_uuid = $1 AND status IN ('invalid', 'succeeded', 'failed', 'down', 'nacked')"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	defer stmt.Close()
	err = stmt.QueryRow(s.RunID).Scan(&c)
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return gerr
	}
	if c == len(s.NodeNames) {
		s.Status = "complete"
		s.save()
	}

	return nil
}

func allShoveyIDsSQL(org *organization.Organization) ([]string, util.Gerror) {
	shoveyList := make([]string, 0)

	sqlStatement := "SELECT run_id FROM goiardi.shoveys WHERE organization_id = $1"

	rows, err := datastore.Dbh.Query(sqlStatement, org.GetId())
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		rows.Close()
		return nil, gerr
	}
	for rows.Next() {
		var runID string
		err = rows.Scan(&runID)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
		shoveyList = append(shoveyList, runID)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	return shoveyList, nil
}

func allShoveysSQL(org *organization.Organization) []*Shovey {
	shoveys := make([]*Shovey, 0)
	sqlStatement := "SELECT run_id, ARRAY(SELECT node_name FROM goiardi.shovey_runs WHERE shovey_uuid = goiardi.shoveys.run_id), command, created_at, updated_at, status, timeout, quorum FROM goiardi.shoveys WHERE organization_id = $1"

	stmt, err := datastore.Dbh.Prepare(sqlStatement)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	rows, err := stmt.Query(org.GetId())
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		s := new(Shovey)
		err = s.fillShoveyFromSQL(rows)
		if err != nil {
			panic(err)
		}
		shoveys = append(shoveys, s)
	}
	return shoveys
}

func (sr *ShoveyRun) addStreamOutSQL(output string, outputType string, seq int, isLast bool) util.Gerror {
	sqlStatement := "INSERT INTO goiardi.shovey_run_streams (shovey_run_id, seq, output_type, output, is_last, created_at) VALUES ($1, $2, $3, $4, $5, NOW())"
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	_, err = tx.Exec(sqlStatement, sr.ID, seq, outputType, output, isLast)
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		tx.Rollback()
		return gerr
	}
	tx.Commit()
	return nil
}

func (sr *ShoveyRun) getStreamOutSQL(outputType string, seq int) ([]*ShoveyRunStream, util.Gerror) {
	var streams []*ShoveyRunStream

	sqlStatement := "SELECT sr.shovey_uuid, sr.node_name, seq, output_type, streams.output, is_last, created_at FROM goiardi.shovey_run_streams streams JOIN goiardi.shovey_runs sr ON streams.shovey_run_id = sr.id WHERE shovey_run_id = $1 AND output_type = $2 AND seq >= $3"

	rows, err := datastore.Dbh.Query(sqlStatement, sr.ID, outputType, seq)
	if err != nil {
		gerr := util.CastErr(err)
		if err == sql.ErrNoRows {
			gerr.SetStatus(http.StatusNotFound)
		} else {
			gerr.SetStatus(http.StatusInternalServerError)
		}
		return nil, gerr
	}
	for rows.Next() {
		srs := new(ShoveyRunStream)
		err = srs.fillShoveyRunStreamFromSQL(rows)
		if err != nil {
			gerr := util.CastErr(err)
			gerr.SetStatus(http.StatusInternalServerError)
			return nil, gerr
		}
		streams = append(streams, srs)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return nil, gerr
	}
	return streams, nil
}

// This is a maybe function now, but it may well work way better to combine the
// stream output in the db. Can't do it in in-mem mode, but that doesn't mean
// we have to do it the same way.
func (sr *ShoveyRun) combineStreamOutSQL(outputType string, seq int) (string, util.Gerror) {

	return "", nil
}

func (s *Shovey) importSaveSQL() error {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	sqlStatement := "INSERT INTO goiardi.shoveys (run_id, command, status, timeout, quorum, created_at, updated_at, organization_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)"

	_, err = tx.Exec(sqlStatement, s.RunID, s.Command, s.Status, s.Timeout, s.Quorum, s.CreatedAt, s.UpdatedAt, s.org.GetId())
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (srs *ShoveyRunStream) importSaveSQL() error {
	s, gerr := Get(srs.org, srs.ShoveyUUID)
	if gerr != nil {
		return gerr
	}
	sr, gerr := s.GetRun(srs.NodeName)
	if gerr != nil {
		return gerr
	}

	tx, err := datastore.Dbh.Begin()
	if err != nil {
		return err
	}
	sqlStatement := "INSERT INTO goiardi.shovey_run_streams (shovey_run_id, seq, output_type, output, is_last, created_at) VALUES ($1, $2, $3, $4, $5, $6)"

	_, err = tx.Exec(sqlStatement, sr.ID, srs.Seq, srs.OutputType, srs.Output, srs.IsLast, srs.CreatedAt)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
