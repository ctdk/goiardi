/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package cookbook

import (
	"encoding/gob"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/ctdk/goiardi/filestore"
)

type constraintTest struct {
	constraint         string
	expectedVersion    string
	expectedNumResults int
}

const minimalCookPath string = "./minimal-cook.json"
const minimal110CookPath string = "./minimal-cook-1.1.0.json"

func TestLatestConstrained(t *testing.T) {
	cbname := "minimal"
	cb, _ := New(cbname)

	// "upload" files - make fake filestore entries
	u := new(filestore.FileStore)
	gob.Register(u)
	c := new(Cookbook)
	gob.Register(c)
	v := new(CookbookVersion)
	gob.Register(v)
	rm := make(map[string]interface{})
	gob.Register(rm)

	a := []string{"0ab75b43c726c3e7c00d7950dd6c3577", "b43166991a65cc7e711a018b93105544", "e2ff77580f69d7612e6a67640fdc2fe0", "5822b0e3808ed57308a0eff8b61f7dc2"}
	var data []byte
	for _, chk := range a {
		f := &filestore.FileStore{Chksum: chk, Data: &data}
		err := f.Save()
		if err != nil {
			t.Error(err)
		}
	}

	mc, err := loadCookbookFromJSON(minimalCookPath)
	if err != nil {
		t.Error(err)
	}

	if _, cerr := cb.NewVersion("1.0.0", mc); cerr != nil {
		t.Error(cerr)
	}

	// and one more cookbook version
	mc2, err := loadCookbookFromJSON(minimal110CookPath)
	if err != nil {
		t.Error(err)
	}

	if _, cerr := cb.NewVersion("1.1.0", mc2); cerr != nil {
		t.Error(cerr)
	}

	conTests := []*constraintTest{
		&constraintTest{"= 1.0.0", "1.0.0", 1},
		&constraintTest{"= 1.1.0", "1.1.0", 1},
		&constraintTest{"~> 1.0.0", "1.0.0", 1},
		&constraintTest{"~> 1.1.0", "1.1.0", 1},
		&constraintTest{"< 1.1.0", "1.0.0", 1},
		&constraintTest{"= 0.1.0", "0.1.0", 0},
		&constraintTest{"> 1.1.0", "1.0.0", 0},
	}

	for _, tc := range conTests {
		tcb := cb.ConstrainedInfoHash("1", tc.constraint)
		vs := tcb["versions"].([]interface{})
		lvs := len(vs)

		if lvs != tc.expectedNumResults {
			t.Errorf("Expected %d results from cb.ConstrainedInfoHash for '%s', but got %d instead.", tc.expectedNumResults, tc.constraint, lvs)
			continue
		}
		if lvs > 0 {
			tcbv := vs[0].(map[string]string)["version"]
			if tcbv != tc.expectedVersion {
				t.Errorf("Expected version '%s' to be returned by cb.ConstrainedInfoHash for '%s', but got '%s'.", tc.expectedVersion, tc.constraint, tcbv)
			}
		}
	}
}

// loadCookbookFromJSON parses json and create a cookbook from it.
func loadCookbookFromJSON(path string) (CookbookVersion, error) {
	jsonByte, err := ioutil.ReadFile(path)
	if err != nil {
		return CookbookVersion{}, err
	}

	var cookbookVersion CookbookVersion
	if err = json.Unmarshal(jsonByte, &cookbookVersion); err != nil {
		return CookbookVersion{}, err
	}
	return cookbookVersion, nil
}
