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

// Package sandbox allows checking files before re-uploading them, so any given
// version of a file need only be uploaded once rather than being uploaded
// repeatedly.
package sandbox

import (
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"time"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
)

/* The structure of the sandbox responses is... inconsistent. */

// Sandbox is a slice of checksums of files, marked completed once they've all
// been uploaded or if they've already been uploaded.
type Sandbox struct {
	ID           string
	CreationTime time.Time
	Completed    bool
	Checksums    []string
	org          *organization.Organization
}

/* We actually generate the sandboxID ourselves, so we don't pass that in. */

// New creates a new sandbox, given a map of null values with file checksums as
// keys.
func New(org *organization.Organization, checksumHash map[string]interface{}) (*Sandbox, error) {
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
	var sandboxID string
	var err error
	for i := 0; i < 5; i++ {
		sandboxID, err = generateSandboxID()
		if err != nil {
			/* Something went very wrong. */
			return nil, err
		}
		if s, _ := Get(org, sandboxID); s != nil {
			logger.Infof("Collision! Somehow %s already existed as a sandbox id on attempt %d. Trying again.", sandboxID, i)
			sandboxID = ""
		}
	}

	if sandboxID == "" {
		err = fmt.Errorf("Somehow every attempt to create a unique sandbox id failed. Bailing.")
		return nil, err
	}
	checksums := make([]string, len(checksumHash))
	j := 0
	for k := range checksumHash {
		checksums[j] = k
		j++
	}

	sbox := &Sandbox{
		ID:           sandboxID,
		CreationTime: time.Now(),
		Completed:    false,
		Checksums:    checksums,
		org:          org,
	}
	return sbox, nil
}

func generateSandboxID() (string, error) {
	randnum := 20
	b := make([]byte, randnum)
	n, err := io.ReadFull(rand.Reader, b)
	if n != len(b) || err != nil {
		return "", err
	}
	idMD5 := md5.New()
	idMD5.Write(b)
	sandboxID := fmt.Sprintf("%x", idMD5.Sum(nil))
	return sandboxID, nil
}

// Get a sandbox.
func Get(org *organization.Organization, sandboxID string) (*Sandbox, error) {
	var sandbox *Sandbox
	var found bool

	if config.UsingDB() {
		var err error
		sandbox, err = getSQL(sandboxID)
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
		ds := datastore.New()
		var s interface{}
		s, found = ds.Get(org.DataKey("sandbox"), sandboxID)
		if s != nil {
			sandbox = s.(*Sandbox)
			sandbox.org = org
		}
	}

	if !found {
		err := fmt.Errorf("Sandbox %s not found", sandboxID)
		return nil, err
	}
	return sandbox, nil
}

// Save the sandbox.
func (s *Sandbox) Save() error {
	if config.Config.UseMySQL {
		if err := s.saveMySQL(); err != nil {
			return err
		}
	} else if config.Config.UsePostgreSQL {
		if err := s.savePostgreSQL(); err != nil {
			return err
		}
	} else {
		ds := datastore.New()
		ds.Set(s.org.DataKey("sandbox"), s.ID, s)
	}
	return nil
}

// Delete a sandbox.
func (s *Sandbox) Delete() error {
	if config.UsingDB() {
		if err := s.deleteSQL(); err != nil {
			return nil
		}
	} else {
		ds := datastore.New()
		ds.Delete(s.org.DataKey("sandbox"), s.ID)
	}
	return nil
}

// GetList returns a list of the ids of all the sandboxes on the system.
func GetList(org *organization.Organization) []string {
	var sandboxList []string
	if config.UsingDB() {
		sandboxList = getListSQL()
	} else {
		ds := datastore.New()
		sandboxList = ds.GetList(org.DataKey("sandbox"))
	}
	return sandboxList
}

// UploadChkList builds the list of file checksums and whether or not they need
// to be uploaded. If they do, the upload URL is also provided.
func (s *Sandbox) UploadChkList() map[string]map[string]interface{} {
	/* Uh... */
	chksumStats := make(map[string]map[string]interface{})
	for _, chk := range s.Checksums {
		chksumStats[chk] = make(map[string]interface{})
		k, _ := filestore.Get(s.org.Name, chk)
		if k != nil {
			chksumStats[chk]["needs_upload"] = false
		} else {
			itemURL := util.JoinStr("/organizations/", s.org.Name, "/file_store/", chk)
			chksumStats[chk]["url"] = util.CustomURL(itemURL)
			chksumStats[chk]["needs_upload"] = true
		}

	}
	return chksumStats
}

// IsComplete returns true if the sandbox is complete.
func (s *Sandbox) IsComplete() error {
	for _, chk := range s.Checksums {
		k, _ := filestore.Get(s.org.Name, chk)
		if k == nil {
			err := fmt.Errorf("Checksum %s not uploaded yet, %s not complete, cannot commit yet.", chk, s.ID)
			return err
		}
	}
	return nil
}

// GetName returns the sandbox's id.
func (s *Sandbox) GetName() string {
	return s.ID
}

// URLType returns the base element of a sandbox's URL.
func (s *Sandbox) URLType() string {
	return "sandboxes"
}

func (s *Sandbox) ContainerType() string {
	return s.URLType()
}

func (s *Sandbox) ContainerKind() string {
	return "containers"
}

// OrgName returns the organization this sandbox belongs to.
func (s *Sandbox) OrgName() string {
	return s.org.Name
}

// AllSandboxes returns all sandboxes on the server.
func AllSandboxes(org *organization.Organization) []*Sandbox {
	var sandboxes []*Sandbox
	if config.UsingDB() {
		sandboxes = allSandboxesSQL()
	} else {
		sandboxList := GetList(org)
		sandboxes = make([]*Sandbox, 0, len(sandboxList))
		for _, s := range sandboxList {
			sb, err := Get(org, s)
			if err != nil {
				continue
			}
			sandboxes = append(sandboxes, sb)
		}
	}
	return sandboxes
}
