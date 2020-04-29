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

// General functions for goiardi database connections, if running in that mode.
// Database engine specific functions are in their respective source files.

package datastore

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/lib/pq"
	"strings"
)

// Dbh is the database handle, shared around.
var Dbh *sql.DB

// Dbhandle is an interface for db handle types that can execute queries.
type Dbhandle interface {
	Prepare(query string) (*sql.Stmt, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// ResRow is an interface for rows returned by Query, or a single row returned by
// QueryRow. Used for passing in a db handle or a transaction to a function.
type ResRow interface {
	Scan(dest ...interface{}) error
}

// For when we need to select a potentially null array of int64s. Moved here
// from groups because we may also need it for policy groups.
type NullInt64Array struct {
	Int64s []int64
	Valid  bool
}

func (n *NullInt64Array) Scan(value interface{}) error {
	if value == nil {
		n.Int64s, n.Valid = nil, false
		return nil
	}
	n.Valid = true
	return pq.Array(&n.Int64s).Scan(value)
}

func (n *NullInt64Array) val() []int64 {
	if n.Valid {
		return n.Int64s
	}
	return make([]int64, 0)
}

// ConnectDB connects to a database with the database name and a map of
// connection options. Currently supports PostgreSQL.
func ConnectDB(dbEngine string, params interface{}) (*sql.DB, error) {
	switch strings.ToLower(dbEngine) {
	case "postgres":
		var connectStr string
		var cerr error
		switch strings.ToLower(dbEngine) {
		case "postgres":
			// no error needed at this step with
			// postgres
			connectStr = formatPostgresqlConStr(params)
		}
		if cerr != nil {
			return nil, cerr
		}
		db, err := sql.Open(strings.ToLower(dbEngine), connectStr)
		if err != nil {
			return nil, err
		}
		if err = db.Ping(); err != nil {
			return nil, err
		}
		db.SetMaxIdleConns(config.Config.DbPoolSize)
		db.SetMaxOpenConns(config.Config.MaxConn)
		return db, nil
	default:
		err := fmt.Errorf("cannot connect to database: unsupported database type %s", dbEngine)
		return nil, err
	}
}

// EncodeToJSON encodes an object to a JSON string.
func EncodeToJSON(obj interface{}) (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	var err error
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Something went wrong with encoding an object to a JSON string for storing.")
		}
	}()
	err = enc.Encode(obj)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// EncodeBlob encodes a slice or map of goiardi object data to save in the
// database. Pass the object to be encoded in like
// datastore.EncodeBlob(&foo.Thing).
func EncodeBlob(obj interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	var err error
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Something went wrong encoding an object for storing in the database")
		}
	}()
	err = enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeBlob decodes the data encoded with EncodeBlob that was stored in the
// database so it can be loaded back into a goiardi object. The 'obj' in the
// arguments *must* be the address of the object receiving the blob of data (e.g.
// datastore.DecodeBlob(data, &obj).
func DecodeBlob(data []byte, obj interface{}) error {
	// hmmm
	dbuf := bytes.NewBuffer(data)
	dec := json.NewDecoder(dbuf)
	dec.UseNumber()
	err := dec.Decode(obj)
	if err != nil {
		return err
	}
	return nil
}

// CheckForOne object of the given type identified by the given name. For this
// function to work, the underlying table MUST have its primary text identifier
// be called "name" and have an "organization_id" column connecting it to its
// parent org.
func CheckForOne(dbhandle Dbhandle, kind string, organization_id int64, name string) (int32, error) {
	var objID int32
	prepStatement := fmt.Sprintf("SELECT id FROM goiardi.%s WHERE name = $1 AND organization_id = $2", kind)
	stmt, err := dbhandle.Prepare(prepStatement)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	err = stmt.QueryRow(name, organization_id).Scan(&objID)
	return objID, err
}
