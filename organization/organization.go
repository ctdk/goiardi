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

package organization

import (
	"bytes"
	"github.com/codeskyblue/go-uuid"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
)

type Organization struct {
	Name string `json:"name"`
	FullName string `json:"full_name"`
	GUID string `json:"guid"`
	uuID uuid.UUID
	id int
}

type privOrganization struct {
	Name *string
	FullName *string
	GUID *string
	UUID *uuid.UUID
	ID *int
}

func New(name, fullName string) (*Organization, util.Gerror) {

}

func Get(orgName string) (Organization, util.Gerror) {

}


func (o *Organization) CheckActor(opUser actor.Actor) util.Gerror {

}

func (o *Organization) Save() util.Gerror {

}

func (o *Organization) Delete() util.Gerror {

}

/* Hmm. Orgs themselves don't have much that needs updating, but it'll get more
 * interesting when RBAC comes along.
 *
 * TODO: Come back soon and investigate.
 */

func GetList() []string {

}

func AllOrganizations() []*Organization {

}

func (o *Organization) export() *privOrganization {
	return &privOrganization{ Name: &o.Name, FullName: &o.FullName, GUID: &o.GUID, UUID: &o.uuID, ID: &o.id }
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
	prv := r.export()
	buf := bytes.NewReader(b)
	encoder := gob.NewDecoder(buf)
	err := encoder.Decode(prv)
	if err != nil {
		return err
	}

	return nil
}
