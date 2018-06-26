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

package acl

import (
	"encoding/gob"
	"fmt"
	"github.com/casbin/casbin"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"io/ioutil"
	"testing"
)

var pivotal *user.User

func init() {
	gob.Register(new(organization.Organization))
	gob.Register(new(user.User))
	gob.Register(new(association.Association))
	gob.Register(new(association.AssociationReq))
	gob.Register(new(group.Group))
	indexer.Initialize(config.Config)
	config.Config.UseAuth = true
	var err error
	confDir, err := ioutil.TempDir("", "acl-test")
	if err != nil {
		panic(err)
	}
	config.Config.PolicyRoot = confDir
}

func TestInitACL(t *testing.T) {
	u, _ := user.New("pivotal")
	u.Admin = true
	u.Save()
	pivotal = u
	org, _ := organization.New("florp", "mlorph normph")
	group.MakeDefaultGroups(org)

	m := casbin.NewModel(modelDefinition)
	e, err := initializeACL(org, m)
	if err != nil {
		t.Error(err)
	}

	z := e.HasPermissionForUser("test1", "groups", "containers", "default", "create", "allow")
	fmt.Printf("z is? %v\n", z)
	
	q := e.Enforce("pivotal", "groups", "containers", "default", "create", "allow")
	fmt.Printf("q is? %v\n", q)

	h := e.Enforce("test1", "clients", "containers", "default", "read", "allow")
	fmt.Printf("h is? %v\n", h)
	
}
