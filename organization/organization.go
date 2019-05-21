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
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/aclhelper"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
	"github.com/pborman/uuid"
	"net/http"
	"os"
	"path"
)

type Organization struct {
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	GUID      string `json:"guid"`
	uuID      uuid.UUID
	id        int
	PermCheck aclhelper.PermChecker `json:"-"`
}

type privOrganization struct {
	Name     *string
	FullName *string
	GUID     *string
	UUID     *uuid.UUID
	ID       *int
}

const searchSchemaSkel = "goiardi_search_%d"

func New(name, fullName string) (*Organization, util.Gerror) {
	var found bool
	uuID := uuid.NewRandom()
	if config.UsingDB() {

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

	o := &Organization{Name: name, FullName: fullName, GUID: guid, uuID: uuID}
	err := o.Save()
	if err != nil {
		return nil, err
	}
	indexer.CreateOrgDex(o.Name)
	return o, nil
}

func Get(orgName string) (*Organization, util.Gerror) {
	var org *Organization
	var found bool
	if config.UsingDB() {

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

	}
	ds := datastore.New()
	ds.Set("organization", o.Name, o)
	return nil
}

func (o *Organization) Delete() util.Gerror {
	if o.Name == "default" {
		return util.Errorf("Cannot delete 'default' organization")
	}
	if config.UsingDB() {

	}
	ds := datastore.New()
	ds.Delete("organization", o.Name)
	err := indexer.DeleteOrgDex(o.Name)
	if err != nil {
		return util.CastErr(err)
	}
	_, aerr := o.PermCheck.DeleteItemACL(o)
	if aerr != nil {
		return util.CastErr(aerr)
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

/* Hmm. Orgs themselves don't have much that needs updating, but it'll get more
 * interesting when RBAC comes along.
 *
 * TODO: Come back soon and investigate.
 */

func GetList() []string {
	var orgList []string
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		orgList = ds.GetList("organization")
	}
	return orgList
}

func AllOrganizations() ([]*Organization, error) {
	if config.UsingDB() {

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
	return &privOrganization{Name: &o.Name, FullName: &o.FullName, GUID: &o.GUID, UUID: &o.uuID, ID: &o.id}
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
//
// At the moment that's still "goiardi", but the idea is that soon organizations
// will have their search tables in separate schemas. The reason for this is to
// hopefully keep them a little more manageable.
//
// Also, while it's pretty obvious this is only especially useful when using
// postgres search.
func (o *Organization) SearchSchemaName() string {
	//return fmt.Sprintf(searchSchemaSkel, o.id)
	return "goiardi"
}

func (o *Organization) GetId() int {
	return o.id
}
