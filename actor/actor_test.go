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

// Tests for the Actor interface (clients and users, in other words)
package actor

import (
	"testing"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
)

func TestActorClient(t *testing.T) {
	config.Config.UseAuth = true
	c, _ := client.New("fooclient")
	c.Save()
	c1, err := GetReqUser("fooclient")
	if err != nil {
		t.Errorf(err.Error())
	}
	y := c1.IsSelf(c)
	if y == false {
		t.Errorf("self not equal to self")
	}
	c2, _ := client.New("foo2client")
	y = c1.IsSelf(c2)
	if y != false {
		t.Errorf("client %s was equal to client %s, but should not have been", c1.GetName(), c2.Name)
	}

	u, _ := user.New("foouser")
	u.Save()

	y = c1.IsSelf(u)
	if y != false {
		t.Errorf("client %s was equal to user %s, but should not have been", c1.GetName(), u.Username)
	}

	c.Delete()
	c.Save()
	c2.Delete()
	c2.Save()
	u.Delete()
	u.Save()
}

func TestActorUser(t *testing.T) {
	config.Config.UseAuth = true
	u, _ := user.New("foouser")
	u.Save()
	u1, err := GetReqUser("foouser")
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

	c, _ := client.New("fooclient")
	c.Save()

	y = u1.IsSelf(c)
	if y != false {
		t.Errorf("user %s was equal to client %s, but should not have been", u1.GetName(), c.Name)
	}

	u.Delete()
	u.Save()
	u2.Delete()
	u2.Save()
	c.Delete()
	c.Save()
}
