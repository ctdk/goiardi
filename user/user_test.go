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

// Now that users are getting split off from clients, they need their own set
// of tests.
package user

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"
)

func TestNewUser(t *testing.T) {
	u, err := New("foo")
	if err != nil {
		t.Errorf(err.Error())
	}
	if u.Username != "foo" {
		t.Errorf("Somehow the username was %s instead of 'foo'", u.Username)
	}
}

func TestSetPasswd(t *testing.T) {
	c, _ := New("foo")
	pass := "abc123"
	tooShort := "123"
	err := c.SetPasswd(tooShort)
	if err == nil {
		t.Errorf("Should not have set a password less than 6 characters, but it did")
	}
	err = c.SetPasswd(pass)
	if err != nil {
		t.Errorf("Should have allowed %s as a password, but didn't", pass)
	}
	err = c.CheckPasswd("abc123")
	if err != nil {
		t.Errorf("abc123 should have been accepted as a password, but it wasn't")
	}
	err = c.CheckPasswd("badpass")
	if err == nil {
		t.Errorf("badpass should not have been accepted, but it was")
	}
}

func TestGobEncodeDecode(t *testing.T) {
	c, _ := New("footged")
	saved := new(bytes.Buffer)
	var err error
	enc := gob.NewEncoder(saved)
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Something went wrong encoding the data store with Gob")
		}
	}()
	err = enc.Encode(c)
	if err != nil {
		t.Errorf(err.Error())
	}
	dec := gob.NewDecoder(saved)
	c2 := new(User)
	err = dec.Decode(&c2)
	if err != nil {
		t.Errorf(err.Error())
	}
	if c2.Username != c.Username {
		t.Errorf("saved user doesn't seem to be equal to original: %v vs %v", c2, c)
	}
}
