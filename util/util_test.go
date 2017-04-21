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

package util

import (
	"net/http"
	"testing"
)

type testObj struct {
	Name        string                 `json:"name"`
	TestURLType string                 `json:"test_url_type"`
	Normal      map[string]interface{} `json:"normal"`
	RunList     []string               `json:"run_list"`
}

func (to *testObj) GetName() string {
	return to.Name
}

func (to *testObj) URLType() string {
	return to.TestURLType
}

func (to *testObj) OrgName() string {
	return "default"
}

// The strange URLs are because the config doesn't get parsed here, so it ends
// up using the really-really default settings.

func TestObjURL(t *testing.T) {
	obj := &testObj{Name: "foo", TestURLType: "bar"}
	url := ObjURL(obj)
	expectedURL := "http://:0/organizations/default/bar/foo"
	if url != expectedURL {
		t.Errorf("expected %s, got %s", expectedURL, url)
	}
}

func TestCustomObjURL(t *testing.T) {
	obj := &testObj{Name: "foo", TestURLType: "bar"}
	url := CustomObjURL(obj, "/baz")
	expectedURL := "http://:0/organizations/default/bar/foo/baz"
	if url != expectedURL {
		t.Errorf("expected %s, got %s", expectedURL, url)
	}
}

func TestCustomURL(t *testing.T) {
	initURL := "/foo/bar"
	url := CustomURL(initURL)
	expectedURL := "http://:0/foo/bar"
	if url != expectedURL {
		t.Errorf("expected %s, got %s", expectedURL, url)
	}
	initURL = "foo/bar"
	url = CustomURL(initURL)
	if url != expectedURL {
		t.Errorf("expected %s, got %s", expectedURL, url)
	}
}

func TestGerror(t *testing.T) {
	errmsg := "foo bar"
	err := Errorf(errmsg)
	if err.Error() != errmsg {
		t.Errorf("expected %s to match %s", err.Error(), errmsg)
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("err.Status() did not return expected default")
	}
	err.SetStatus(http.StatusNotFound)
	if err.Status() != http.StatusNotFound {
		t.Errorf("SetStatus did not set Status correctly")
	}
}

func TestFlatten(t *testing.T) {
	rl := []string{"recipe[foo]", "role[bar]"}
	normmap := make(map[string]interface{})
	normmap["foo"] = "bar"
	normmap["baz"] = "buz"
	normmap["slice"] = []string{"fee", "fie", "fo"}
	normmap["map"] = make(map[string]interface{})
	normmap["map"].(map[string]interface{})["first"] = "mook"
	normmap["map"].(map[string]interface{})["second"] = "nork"
	obj := &testObj{Name: "foo", TestURLType: "bar", RunList: rl, Normal: normmap}
	flattened := FlattenObj(obj)
	if _, ok := flattened["name"]; !ok {
		t.Errorf("obj name was not flattened correctly")
	}
	if flattened["name"].(string) != obj.Name {
		t.Errorf("flattened name not correct, wanted %s got %v", obj.Name, flattened["name"])
	}
	if _, ok := flattened["foo"]; !ok {
		t.Errorf("Foo should have been set, but it wasn't")
	}
	if _, ok := flattened["normal"]; ok {
		t.Errorf("The 'normal' field was set, but shouldn't have been.")
	}
	if _, ok := flattened["map_first"]; !ok {
		t.Errorf("normal -> map -> second should have been flattened to map_second, but it wasn't")
	}
	if r, ok := flattened["recipe"]; ok {
		if r.([]string)[0] != "foo" {
			t.Errorf("recipe list should have included foo, but it had %v instead", r.([]string)[0])
		}
	} else {
		t.Errorf("No recipe list")
	}
	if r, ok := flattened["role"]; ok {
		if r.([]string)[0] != "bar" {
			t.Errorf("role list should have included bar, but it had %v instead", r.([]string)[0])
		}
	} else {
		t.Errorf("No role list")
	}
}

func TestMapify(t *testing.T) {
	rl := []string{"recipe[foo]", "role[bar]"}
	normmap := make(map[string]interface{})
	normmap["foo"] = "bar"
	normmap["baz"] = "buz"
	normmap["slice"] = []string{"fee", "fie", "fo"}
	normmap["map"] = make(map[string]interface{})
	normmap["map"].(map[string]interface{})["first"] = "mook"
	normmap["map"].(map[string]interface{})["second"] = "nork"
	obj := &testObj{Name: "foo", TestURLType: "bar", RunList: rl, Normal: normmap}
	mapify := MapifyObject(obj)
	if mapify["name"].(string) != obj.Name {
		t.Errorf("Mapify names didn't match, expecte %s, got %v", obj.Name, mapify["name"])
	}
	if _, ok := mapify["normal"]; !ok {
		t.Errorf("There should have been a normal key for the map")
	}
	if _, ok := mapify["foo"]; ok {
		t.Errorf("There was a foo key in mapify, and there should not have been.")
	}
}

func TestIndexify(t *testing.T) {
	rl := []string{"recipe[foo]", "role[bar]"}
	normmap := make(map[string]interface{})
	normmap["foo"] = "bar"
	normmap["baz"] = "buz"
	normmap["slice"] = []string{"fee", "fie", "fo"}
	normmap["map"] = make(map[string]interface{})
	normmap["map"].(map[string]interface{})["first"] = "mook"
	normmap["map"].(map[string]interface{})["second"] = "nork"
	obj := &testObj{Name: "foo", TestURLType: "bar", RunList: rl, Normal: normmap}
	flatten := FlattenObj(obj)
	indexificate := Indexify(flatten)
	if indexificate[0] != "baz:buz" {
		t.Errorf("The first element of the indexified object should have been 'baz:buz', but instead it was %s", indexificate[0])
	}
}

func TestValidateName(t *testing.T) {
	goodName := "foo-bar.baz"
	badName := "FAh!!"
	if !ValidateName(goodName) {
		t.Errorf("%s should have passed name validation, but didn't", goodName)
	}
	if ValidateName(badName) {
		t.Errorf("%s should not have passed name validation, but somehow did", badName)
	}
}

func TestValidateUserName(t *testing.T) {
	goodName := "foo"
	badName := "USERNAME"
	if !ValidateUserName(goodName) {
		t.Errorf("%s should have passed user name validation, but didn't", goodName)
	}
	if ValidateUserName(badName) {
		t.Errorf("%s should not have passed user name validation, but somehow did", badName)
	}
}

func TestValidateDBagName(t *testing.T) {
	goodName := "foo-bar"
	badName := "FaH!!"
	if !ValidateName(goodName) {
		t.Errorf("%s should have passed data bag name validation, but didn't", goodName)
	}
	if ValidateName(badName) {
		t.Errorf("%s should not have passed data bag name validation, but somehow did", badName)
	}
}

func TestValidateEnvName(t *testing.T) {
	goodName := "foo-bar"
	badName := "FAh!!"
	if !ValidateName(goodName) {
		t.Errorf("%s should have passed env name validation, but didn't", goodName)
	}
	if ValidateName(badName) {
		t.Errorf("%s should not have passed env name validation, but somehow did", badName)
	}
}

// A lot of the validations get taken care of with chef pedant, honestly
func TestValidateAsVersion(t *testing.T) {
	goodVersion := "1.0.0"
	goodVersion2 := "1.0"
	badVer1 := "1"
	badVer2 := "foo"
	var badVer3 interface{}
	badVer3 = nil

	if _, err := ValidateAsVersion(goodVersion); err != nil {
		t.Errorf("%s should have passed version validation, but didn't", goodVersion)
	}
	if _, err := ValidateAsVersion(goodVersion2); err != nil {
		t.Errorf("%s should have passed version validation, but didn't", goodVersion2)
	}
	if _, err := ValidateAsVersion(badVer1); err == nil {
		t.Errorf("%s should not have passed version validation, but did", badVer1)
	}
	if _, err := ValidateAsVersion(badVer2); err == nil {
		t.Errorf("%s should not have passed version validation, but did", badVer2)
	}
	if v, err := ValidateAsVersion(badVer3); err != nil {
		t.Errorf("nil should have passed version validation, but did")
	} else if v != "0.0.0" {
		t.Errorf("Should have come back as 0.0.0, but it came back as %v", v)
	}
}
