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

package shovey

/* PostgreSQL funcs for shovey */

import (
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"github.com/lib/pq"
	"net/http"
	"time"
)

func (s *Shovey) fillShoveyFromPostgreSQL(row datastore.ResRow) error {
	var ca, ua pq.NullTime
	var nn util.StringSlice
	var tm int64
	err := row.Scan(&s.RunID, &nn, &s.Command, &ca, &ua, &s.Status, &tm, &s.Quorum)
	if err != nil {
		return err
	}
	if ca.Valid {
		s.CreatedAt = ca.Time
	}
	if ua.Valid {
		s.UpdatedAt = ua.Time
	}
	s.Timeout = time.Duration(tm)

	s.NodeNames = nn

	return nil
}

func (sr *ShoveyRun) fillShoveyRunFromPostgreSQL(row datastore.ResRow) error {
	var at, et pq.NullTime
	err := row.Scan(&sr.ID, &sr.ShoveyUUID, &sr.NodeName, &sr.Status, &at, &et, &sr.Error, &sr.ExitStatus)
	if err != nil {
		return err
	}
	if at.Valid {
		sr.AckTime = at.Time
	}
	if et.Valid {
		sr.EndTime = et.Time
	}
	return nil
}

func (srs *ShoveyRunStream) fillShoveyRunStreamFromPostgreSQL(row datastore.ResRow) error {
	var ca pq.NullTime
	err := row.Scan(&srs.ShoveyUUID, &srs.NodeName, &srs.Seq, &srs.OutputType, &srs.Output, &srs.IsLast, &ca)
	if err != nil {
		return err
	}
	if ca.Valid {
		srs.CreatedAt = ca.Time
	}
	return nil
}

func (s *Shovey) savePostgreSQL() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.merge_shoveys($1, $2, $3, $4, $5)", s.RunID, s.Command, s.Status, s.Timeout, s.Quorum)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}

func (sr *ShoveyRun) savePostgreSQL() util.Gerror {
	tx, err := datastore.Dbh.Begin()
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	_, err = tx.Exec("SELECT goiardi.merge_shovey_runs($1, $2, $3, $4, $5, $6, $7)", sr.ShoveyUUID, sr.NodeName, sr.Status, sr.AckTime, sr.EndTime, sr.Error, sr.ExitStatus)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}
	tx.Commit()
	return nil
}
