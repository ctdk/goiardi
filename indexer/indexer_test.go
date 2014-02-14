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

// Package chef_crypto bundles up crytographic routines for goairdi.
package indexer

import (
	"testing"
	"github.com/ctdk/goiardi/util"
	"fmt"
	"os"
	"io/ioutil"
)

type testObj struct {
	Name string `json:"name"`
	UrlType string `json:"url_type"`
	Normal map[string]interface{} `json:"normal"`
	RunList []string `json:"run_list"`
}

func (to *testObj) DocId() string {
	return to.Name
}

func (to *testObj) Index() string {
	return "test_obj"
}

func (to *testObj) Flatten() []string {
	flatten := util.FlattenObj(to)
	indexified := util.Indexify(flatten)
	return indexified
}

func TestIndexObj(t *testing.T){
	obj := &testObj{ Name: "foo", UrlType: "bar" }
	IndexObj(obj)
}

var idxTmpDir = idxTmpGen()

func idxTmpGen() string {
	tm, err := ioutil.TempDir("", "idx-test")
	if err != nil {
		panic("Couldn't create temporary directory!")
	}
	return tm
}

func TestSave(t *testing.T) {
	tmpfile := fmt.Sprintf("%s/idx.bin", idxTmpDir)
	err := SaveIndex(tmpfile)
	if err != nil {
		t.Errorf("Save() gave an error: %s", err)
	}
}

func TestLoad(t *testing.T) {
	tmpfile := fmt.Sprintf("%s/idx.bin", idxTmpDir)
	err := LoadIndex(tmpfile)
	if err != nil {
		t.Errorf("Load() save an error: %s", err)
	}
}



// clean up

func TestCleanup(t *testing.T) {
	os.RemoveAll(idxTmpDir)
}
