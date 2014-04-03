/* Sandbox structs, for testing whether cookbook files need to be uploaded */

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

// Package sandbox allows checking files before re-uploading the, so any given
// version of a file need only be uploaded once rather than being uploaded
// repeatedly.
package sandbox

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/util"
	"fmt"
	"crypto/md5"
	"crypto/rand"
	"io"
	"log"
	"time"
	"database/sql"
)

/* The structure of the sandbox responses is... inconsistent. */

type Sandbox struct {
	Id string
	CreationTime time.Time
	Completed bool
	Checksums []string
}

/* We actually generate the sandbox_id ourselves, so we don't pass that in. */

// Create a new sandbox, given a map of null values with file checksums as keys.
func New(checksum_hash map[string]interface{}) (*Sandbox, error){
	/* For some reason the checksums come in a JSON hash that looks like
	 * this:
 	 * { "checksums": {
	 * "385ea5490c86570c7de71070bce9384a":null,
  	 * "f6f73175e979bd90af6184ec277f760c":null,
  	 * "2e03dd7e5b2e6c8eab1cf41ac61396d5":null
  	 * } } --- per the chef server api docs. Not sure why it comes in that
	 * way rather than as an array, since those nulls are apparently never
	 * anything but nulls. */

	/* First generate an id for this sandbox. Collisions are certainly
	 * possible, so we'll give it five tries to make a unique one before
	 * bailing. This may later turn out not to be the ideal sandbox creation
	 * method, but we'll see. */
	var sandbox_id string
	var err error
	for i := 0; i < 5; i++ {
		sandbox_id, err = generate_sandbox_id()
		if err != nil {
			/* Something went very wrong. */
			return nil, err 
		}
		if s := Get(sandbox_id); s != nil {
			err = fmt.Errorf("Collision! Somehow %s already existed as a sandbox id on attempt %d. Trying again.", sandbox_id, i)
			sandbox_id = ""
			log.Println(err)
		}
	}

	if sandbox_id == "" {
		err = fmt.Errorf("Somehow every attempt to create a unique sandbox id failed. Bailing.")
		return nil, err
	} 
	checksums := make([]string, len(checksum_hash))
	j := 0
	for k, _ := range checksum_hash {
		checksums[j] = k
		j++
	}

	sbox := &Sandbox{
		Id: sandbox_id,
		CreationTime: time.Now(),
		Completed: false,
		Checksums: checksums,
	}
	return sbox, nil
}

func generate_sandbox_id() (string, error) {
	randnum := 20
	b := make([]byte, randnum)
	n, err := io.ReadFull(rand.Reader, b)
	if n != len(b) || err != nil {
		return "", err
	}
	id_md5 := md5.New()
	id_md5.Write(b)
	sandbox_id := fmt.Sprintf("%x", id_md5.Sum(nil))
	return sandbox_id, nil
}

func (s *Sandbox)fillSandboxFromSQL(row *sql.Row) error {
	if config.Config.UseMySQL {
		var csb []byte
		err := row.Scan(&s.Id, &s.CreationTime, &csb, &s.Completed)
		if err != nil {
			return err
		}
		var q interface{}
		q, err = data_store.DecodeBlob(csb, s.Checksums)
		if err != nil {
			return err
		}
		s.Checksums = q.([]string)
	} else {
		err := fmt.Errorf("no database configured, operating in in-memory mode -- fillSandboxFromSQL cannot be run")
		return err
	}
}

func Get(sandbox_id string) (*Sandbox, error){
	var sandbox *Sandbox
	var found bool

	if config.Config.UseMySQL {
		sandbox = new(Sandbox)
		stmt, err := data_store.Dbh.Prepare("SELECT sbox_id, creation_time, checksums, completed FROM sandboxes WHERE sbox_id = ?")
		if err != nil {
			return nil, err
		}
		defer stmt.Close()
		row := stmt.QueryRow(sandbox_id)
		err = sandbox.fillSandboxFromSQL(sandbox_id)
		if err != nil {
			if err == sql.ErrNoRows {
				found = false
			} else {
				return nil, err
			}
		} else {
			found = true
		}
	} else {
		ds := data_store.New()
		var s interface{}
		s, found = ds.Get("sandbox", sandbox_id)
		sandbox = s.(*Sandbox)
	}

	if !found {
		err := fmt.Errorf("Sandbox %s not found", sandbox_id)
		return nil, err
	}
	return sandbox, nil
}

func (s *Sandbox) Save() error {
	if config.Config.UseMySQL {
		ckb, ckerr := data_store.EncodeBlob(s.Checksums)
		if ckerr != nil {
			return ckerr
		}
		tx, err := data_store.Dbh.Begin()
		if err != nil {
			return err
		}
		var sbox_id string
		err = tx.QueryRow(s.Id).Scan(&sbox_id)
		if err == nil {
			_, err = tx.Exec("UPDATE sandboxes SET checksums = ?, completed = ? WHERE sbox_id = ?", ckb, s.Completed, s.Id)
				if err != nil {
					tx.Rollback()
					return err
				}
		} else {
			if err != sql.ErrNoRows {
				tx.Rollback()
				return err
			}
			_, err = tx.Exec("INSERT INTO sandboxes (sbox_id, creation_time, checksums, completed) VALUES (?, ?, ?, ?)", s.Id, s.CreationTime, ckb, s.Completed)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
		tx.Commit()
	} else {
		ds := data_store.New()
		ds.Set("sandbox", s.Id, s)
	}
	return nil
}

func (s *Sandbox) Delete() error {
	if config.Config.UseMySQL {
		tx, err := data_store.Dbh.Begin()
		if err != nil {
			return err
		}
		_, err := tx.Exec("DELETE FROM sandboxes WHERE sbox_id = ?", s.Id)
		if err != nil {
			terr := tx.Rollback()
			if terr != nil {
				err = fmt.Errorf("deleting sandbox %s had an error '%s', and then rolling back the transaction gave another error '%s'", s.Id, err.Error(), terr.Error())
			}
			return err
		}
		tx.Commit()
	} else {
		ds := data_store.New()
		ds.Delete("sandbox", s.Id)
	}
	return nil
}

func GetList() []string {
	var sandbox_list []string
	if config.Config.UseMySQL {
		rows, err := data_store.Dbh.Query("SELECT sbox_id FROM sandboxes")
		if err != nil {
			if err != sql.ErrNoRows {
				log.Fatal(err)
			}
			rows.Close()
			return sandbox_list
		}
		sandbox_list = make([]string, 0)
		for rows.Next() {
			var sbox_id string
			err = rows.Scan(&sbox_id)
			if err != nil {
				log.Fatal(err)
			}
			sandbox_list = append(sandbox_list, sbox_id)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			log.Fatal(err)
		}
	} else {
		ds := data_store.New()
		sandbox_list = ds.GetList("sandbox")
	}
	return sandbox_list
}

// Creates the list of file checksums and whether or not they need to be
// uploaded or not. If they do, the upload URL is also provided.
func (s *Sandbox) UploadChkList() map[string]map[string]interface{} {
	/* Uh... */
	chksum_stats := make(map[string]map[string]interface{})
	for _, chk := range s.Checksums {
		chksum_stats[chk] = make(map[string]interface{})
		k, _ := filestore.Get(chk)
		if k != nil {
			chksum_stats[chk]["needs_upload"] = false
		} else {
			item_url := fmt.Sprintf("/file_store/%s", chk)
			chksum_stats[chk]["url"] = util.CustomURL(item_url)
			chksum_stats[chk]["needs_upload"] = true
		}

	}
	return chksum_stats
}

// Is the sandbox complete?
func (s *Sandbox) IsComplete() error {
	for _, chk := range s.Checksums {
		k, _ := filestore.Get(chk)
		if k == nil {
			err := fmt.Errorf("Checksum %s not uploaded yet, %s not complete, cannot commit yet.", chk, s.Id)
			return err
		}
	}
	return nil
}

func (s *Sandbox) GetName() string {
	return s.Id
}

func (s *Sandbox) URLType() string {
	return "sandboxes"
}
