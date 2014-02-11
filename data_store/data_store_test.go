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

package data_store

import (
	"testing"
)

type dsObj struct {
	Name string `json:"name"`
	JsonClass string `json:"json_class"`
	ChefType string `json:"chef_type"`
}

func makeDsObj() *dsObj {
	return &dsObj{ Name: "baz", JsonClass: "Chef::DsObj", ChefType: "ds_obj" }
}

func TestNew(t *testing.T){
	if d := New(); d == nil {
		t.Errorf("New() should have returned a data store object, but returned nil")
	}
}

func TestSet(t *testing.T){
	ds := New()
	baz := makeDsObj()
	ds.Set("foo", "bar", baz)
}

func TestGet(t *testing.T){
	ds := New()
	val, found := ds.Get("foo", "bar2")
	if found != false {
		t.Errorf("Get() returned a result improperly")
	}
	baz := makeDsObj()
	ds.Set("foo", "bar2", baz)
	val, found = ds.Get("foo", "bar2")
	if found == false {
		t.Errorf("Get() did not return a result properly, got '%v' :: %v", val, found)
	}
}
