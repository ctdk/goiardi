/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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

package organization

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/ctdk/chefcrypto"
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	_ "github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
	"github.com/pborman/uuid"
	"github.com/tideland/golib/logger"
	"net/http"
	"os"
	"path"
	"sync"
)

type Organization struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	GUID          string `json:"guid"`
	uuID          uuid.UUID
	id            int64
	PermCheck     aclhelper.PermChecker `json:"-"`
	shoveyKey     *SigningKeys
}

type privOrganization struct {
	Name     *string
	FullName *string
	GUID     *string
	UUID     *uuid.UUID
	ID       *int64
	ShoveyKey *string
}

// SigningKeys are the public and private keys for signing shovey requests.
type SigningKeys struct {
	m *sync.RWMutex
	PrivKey *rsa.PrivateKey
}

func New(name, fullName string) (*Organization, util.Gerror) {
	var found bool
	uuID := uuid.NewRandom()
	if config.UsingDB() {
		found, _ = checkForOrgSQL(datastore.Dbh, name)
	} else {
		ds := datastore.New()
		_, found = ds.Get("organization", name)
	}
	if found {
		err := util.Errorf("an organization with this name already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	guid := fmt.Sprintf("%32x", []byte(uuID))
	// NOTE: This may require more thorough validation down the line
	if _, sterr := util.ValidateAsString(name); sterr != nil {
		gerr := util.Errorf("organization name invalid or missing")
		return nil, gerr
	}
	if _, sterr := util.ValidateAsString(fullName); sterr != nil {
		gerr := util.Errorf("organization full name invalid or missing")
		return nil, gerr
	}

	// create the filestore dir
	if config.Config.LocalFstoreDir != "" {
		p := path.Join(config.Config.LocalFstoreDir, name)
		err := os.Mkdir(p, os.ModeDir|0700)
		if err != nil && !os.IsExist(err) {
			return nil, util.CastErr(err)
		}
	}

	o := &Organization{Name: name, FullName: fullName, GUID: guid, uuID: uuID,}

	// Create the SigningKeys struct, if needed.
	if config.Config.UseShovey && !config.UsingExternalSecrets() {
		skm := new(sync.RWMutex)
		o.shoveyKey = &SigningKeys{skm, nil}
		if skErr := o.GenerateShoveyKey(); skErr != nil {
			return nil, skErr
		}
	}

	err := o.Save()
	if err != nil {
		return nil, err
	}
	if ierr := indexer.CreateOrgDex(o); ierr != nil {
		logger.Debugf(ierr.Error())
		return nil, util.CastErr(ierr)
	}

	return o, nil
}

func Get(orgName string) (*Organization, util.Gerror) {
	var org *Organization
	var found bool
	if config.UsingDB() {
		var err error
		org, err = getOrgSQL(orgName)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, util.CastErr(err)
			}
			found = false
		} else {
			found = true
		}
	} else {
		ds := datastore.New()
		var o interface{}
		o, found = ds.Get("organization", orgName)
		if o != nil {
			org = o.(*Organization)
		}
	}
	if !found {
		err := util.Errorf("organization '%s' does not exist.", orgName)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return org, nil
}

func (o *Organization) Save() util.Gerror {
	if config.UsingDB() {
		return o.saveSQL()
	}
	ds := datastore.New()
	ds.Set("organization", o.Name, o)
	return nil
}

func (o *Organization) Delete() util.Gerror {
	if o.Name == "default" {
		return util.Errorf("Cannot delete 'default' organization")
	}

	// Delete the ACL first, methinks
	aerr := o.PermCheck.DeletePolicy()
	if aerr != nil {
		return util.CastErr(aerr)
	}

	// Files need to be deleted as well. Hrm.
	if config.UsingDB() {
		if err := o.deleteSQL(); err != nil {
			return util.CastErr(err)
		}
		return nil
	}
	ds := datastore.New()
	ds.Delete("organization", o.Name)
	err := indexer.DeleteOrgDex(o)
	if err != nil {
		return util.CastErr(err)
	}

	return nil
}

func (o *Organization) ToJSON() map[string]interface{} {
	orgJSON := make(map[string]interface{}, 3)
	orgJSON["name"] = o.Name
	orgJSON["full_name"] = o.FullName
	orgJSON["guid"] = o.GUID
	return orgJSON
}

func (o *Organization) DataKey(typeKey string) string {
	return util.JoinStr(typeKey, "-", o.Name)
}

func GetList() []string {
	var orgList []string
	if config.UsingDB() {
		return getListSQL()
	} else {
		ds := datastore.New()
		orgList = ds.GetList("organization")
	}
	return orgList
}

func AllOrganizations() ([]*Organization, error) {
	if config.UsingDB() {
		return allOrgsSQL()
	}
	orgList := GetList()
	orgs := make([]*Organization, 0, len(orgList))
	for _, o := range orgList {
		org, _ := Get(o)
		if org != nil {
			orgs = append(orgs, org)
		}
	}
	return orgs, nil
}

func (o *Organization) export() *privOrganization {
	var shovKey string

	if o.shoveyKey != nil {
		o.shoveyKey.m.RLock()
		defer o.shoveyKey.m.RUnlock()

		if !config.UsingExternalSecrets() && o.shoveyKey.PrivKey != nil {
			// hope for the best, eh
			shovKey, _ = chefcrypto.PrivateKeyToString(o.shoveyKey.PrivKey)
		}
	}

	return &privOrganization{Name: &o.Name, FullName: &o.FullName, GUID: &o.GUID, UUID: &o.uuID, ID: &o.id, ShoveyKey: &shovKey }
}

func (o *Organization) GobEncode() ([]byte, error) {
	prv := o.export()
	buf := new(bytes.Buffer)
	decoder := gob.NewEncoder(buf)
	if err := decoder.Encode(prv); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (o *Organization) GobDecode(b []byte) error {
	prv := o.export()
	buf := bytes.NewReader(b)
	encoder := gob.NewDecoder(buf)
	err := encoder.Decode(prv)
	if err != nil {
		return err
	}

	return nil
}

// TODO: fill these in

func ExportAllOrgs() []map[string]interface{} {
	return nil
}
func Import(orgData map[string]interface{}) (*Organization, error) {
	return nil, nil
}

// Turns out it's more straightforward to do the '$$root$$' perm checks against
// the organization in question. Huh. That means orgs need to satisfy the
// aclhelper interface.

func (o *Organization) GetName() string {
	return o.Name
}

func (o *Organization) ContainerKind() string {
	return "containers"
}

func (o *Organization) ContainerType() string {
	return "$$root$$"
}

func (o *Organization) SetPermCheck(p aclhelper.PermChecker) {
	o.PermCheck = p
}

// SearchSchemaName is a handy little helper method that will return the schema
// that holds the search tables for this organization.
func (o *Organization) SearchSchemaName() string {
	return fmt.Sprintf(util.SearchSchemaSkel, o.id)
}

func (o *Organization) GetId() int64 {
	return o.id
}

// OrgURLBase returns the common "/organizations/<foo>" portion of a Chef
// object's URL.
func (o *Organization) OrgURLBase() string {
	return fmt.Sprintf("/organizations/%s", o.Name)
}

// GenerateShoveyKey generates a new private key for signing shovey requests
// and sets that key in the org object.
func (o *Organization) GenerateShoveyKey() util.Gerror {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusInternalServerError)
		return gerr
	}

	if o.shoveyKey == nil { // make one real quick-like
		skm := new(sync.RWMutex)
		o.shoveyKey = &SigningKeys{skm, nil}
	}

	o.shoveyKey.m.RLock()
	defer o.shoveyKey.m.RUnlock()
	o.shoveyKey.PrivKey = priv
	return nil
}

// ShoveyPrivKey returns this organization's private key for signing shovey
// requests, and an error if the key cannot be found.
func (o *Organization) ShoveyPrivKey() (*rsa.PrivateKey, error) {
	if o.shoveyKey == nil {
		return nil, fmt.Errorf("No private key has been set for organization %s", o.Name)
	}
	o.shoveyKey.m.RLock()
	defer o.shoveyKey.m.RUnlock()

	if o.shoveyKey.PrivKey == nil {
		return nil, fmt.Errorf("No signing key available for org %s!", o.Name)
	}

	// I believe the extra confusing seeming step is to ensure the private
	// key doesn't change from under us. This may not be desirable anymore.
	j := *o.shoveyKey.PrivKey
	return &j, nil
}

// ShoveyPublicKey returns this organization's public key for verifying shovey
// requests, and an error if the key cannot be found.
func (o *Organization) ShoveyPubKey() (string, error) {
	if o.shoveyKey == nil {
		return "", fmt.Errorf("No private key has been set for organization %s, so there's no public key either.", o.Name)
	}
	o.shoveyKey.m.RLock()
	defer o.shoveyKey.m.RUnlock()

	if o.shoveyKey.PrivKey == nil {
		return "", fmt.Errorf("No private key for signing shovey request can be found for org %s, so the public key can not be retrieved either.", o.Name)
	}

	p, err := chefcrypto.PublicKeyToString(o.shoveyKey.PrivKey.PublicKey)
	if err != nil {
		return "", err
	}

	return p, nil
}
