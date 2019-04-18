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

// Tests for the Actor interface (clients and users, in other words)
package actor

import (
	"encoding/gob"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/fakeacl"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"testing"
)

var org *organization.Organization

func init() {
	indexer.Initialize(config.Config)
}

func TestActorClient(t *testing.T) {
	gob.Register(new(organization.Organization))
	org, _ = organization.New("default", "boo")
	fakeacl.LoadFakeACL(org)
	org.Save()

	config.Config.UseAuth = true
	indexer.Initialize(config.Config)

	c, _ := client.New(org, "fooclient")
	gob.Register(c)
	c.Save()
	c1, err := GetReqUser(org, "fooclient")
	if err != nil {
		t.Errorf(err.Error())
	}
	y := c1.IsSelf(c)
	if y == false {
		t.Errorf("self not equal to self")
	}
	c2, _ := client.New(org, "foo2client")
	y = c1.IsSelf(c2)
	if y != false {
		t.Errorf("client %s was equal to client %s, but should not have been", c1.GetName(), c2.Name)
	}

	u, _ := user.New("foouser")
	gob.Register(u)
	u.Save()

	y = c1.IsSelf(u)
	if y != false {
		t.Errorf("client %s was equal to user %s, but should not have been", c1.GetName(), u.Username)
	}

	c.Delete()
	c2.Delete()
	u.Delete()
}

func TestActorUser(t *testing.T) {
	config.Config.UseAuth = true
	u, err := user.New("foo1user")
	if err != nil {
		t.Errorf(err.Error())
	}
	u.Save()
	u1, err := GetReqUser(org, "foo1user")
	if err != nil {
		t.Errorf(err.Error())
	}
	y := u1.IsSelf(u)
	if y == false {
		t.Errorf("self not equal to self")
	}
	u2, _ := user.New("foo2user")
	y = u1.IsSelf(u2)
	if y != false {
		t.Errorf("user %s was equal to user %s, but should not have been", u1.GetName(), u2.Username)
	}

	c, _ := client.New(org, "foo1client")
	c.Save()

	y = u1.IsSelf(c)
	if y != false {
		t.Errorf("user %s was equal to client %s, but should not have been", u1.GetName(), c.Name)
	}

	u.Delete()
	u2.Delete()
	c.Delete()
}
