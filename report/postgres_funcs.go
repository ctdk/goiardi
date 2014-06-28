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

/* PostgreSQL funcs for reports */

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/lib/pq"
)

func (r *Report) fillReportFromPostgreSQL(row data_store.ResRow) error {
	var res, dat []byte
	var st, et pq.NullTime
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

func (r *Report) savePostgreSQL() error {
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
	_, err = tx.Exec("SELECT goiardi.merge_reports($1, $2, $3, $4, $5, $6, $7, $8, $9)", r.RunId, r.NodeName, r.StartTime, r.EndTime, r.TotalResCount, r.Status, r.RunList, res, dat)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
