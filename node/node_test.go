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

// Node tests
package node

import (
	"encoding/gob"
	"testing"
)

func TestActionAtADistance(t *testing.T) {
	n, _ := New("foo2")
	gob.Register(n)
	n.Normal["foo"] = "bar"
	n.Save()
	n2, _ := Get("foo2")
	if n.Name != n2.Name {
		t.Errorf("Node names should have been the same, but weren't, got %s and %s", n.Name, n2.Name)
	}
	if n.Normal["foo"] != n2.Normal["foo"] {
		t.Errorf("Normal attribute 'foo' should have been equal between the two copies of the node, but weren't.")
	}
	n2.Normal["foo"] = "blerp"
	if n.Normal["foo"] == n2.Normal["foo"] {
		t.Errorf("Normal attribute 'foo' should not have been equal between the two copies of the node, but were.")
	}
	n2.Save()
	n3, _ := Get("foo2")
	if n3.Normal["foo"] != n2.Normal["foo"] {
		t.Errorf("Normal attribute 'foo' should have been equal between the two copies of the node after saving a second time, but weren't.")
	}
}
