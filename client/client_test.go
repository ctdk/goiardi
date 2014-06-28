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

// Some client tests, for now.
package client

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"
)

func TestGobEncodeDecode(t *testing.T) {
	c, _ := New("foo")
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
	c2 := new(Client)
	err = dec.Decode(&c2)
	if err != nil {
		t.Errorf(err.Error())
	}
	if c2.Name != c.Name {
		t.Errorf("saved user doesn't seem to be equal to original: %v vs %v", c2, c)
	}
}

func TestActionAtADistance(t *testing.T) {
	c, _ := New("foo2")
	gob.Register(c)
	c.Save()
	c2, _ := Get("foo2")
	if c.Name != c2.Name {
		t.Errorf("Client names should have been the same, but weren't, got %s and %s", c.Name, c2.Name)
	}
	c2.Validator = true
	if c.Validator == c2.Validator {
		t.Errorf("Changing the value of validator on one client improperly changed it on the other")
	}
}
