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

package indexer

import (
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/util"
	"io/ioutil"
	"os"
	"testing"
)

type testObj struct {
	Name    string                 `json:"name"`
	URLType string                 `json:"url_type"`
	Normal  map[string]interface{} `json:"normal"`
	RunList []string               `json:"run_list"`
}

var conf *config.Conf

func init() {
	conf = &config.Conf{}
	idxTmpDir = idxTmpGen()
	conf.IndexFile = fmt.Sprintf("%s/idx.bin", idxTmpDir)
	Initialize(conf)
}

func (to *testObj) DocID() string {
	return to.Name
}

func (to *testObj) Index() string {
	return "test_obj"
}

func (to *testObj) Flatten() map[string]interface{} {
	flatten := util.FlattenObj(to)
	return flatten
}

func TestIndexObj(t *testing.T) {
	obj := &testObj{Name: "foo", URLType: "bar"}
	IndexObj(obj)
}

var idxTmpDir string

func idxTmpGen() string {
	tm, err := ioutil.TempDir("", "idx-test")
	if err != nil {
		panic("Couldn't create temporary directory!")
	}
	return tm
}

func TestSave(t *testing.T) {
	err := SaveIndex()
	if err != nil {
		t.Errorf("Save() gave an error: %s", err)
	}
}

func TestLoad(t *testing.T) {
	err := LoadIndex()
	if err != nil {
		t.Errorf("Load() save an error: %s", err)
	}
}

// clean up

func TestCleanup(t *testing.T) {
	os.RemoveAll(idxTmpDir)
}
