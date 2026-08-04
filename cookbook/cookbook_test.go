/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"testing"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/organization"
)

func init() {
	indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
	gob.Register(new(Cookbook))
	gob.Register(new(CookbookVersion))
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(map[string]string{})
	gob.Register(new(filestore.FileStore))
}

var cookbookOrgCounter int

func setupCookbookTestOrg(t *testing.T) *organization.Organization {
	gob.Register(new(organization.Organization))
	cookbookOrgCounter++
	name := fmt.Sprintf("testcookbook-%d", cookbookOrgCounter)
	org, err := orgloader.New(name, "Test Cookbook Org")
	if err != nil {
		t.Fatalf("failed to create test org: %s", err.Error())
	}
	return org
}

func makeVersionJSON(name, version string) map[string]interface{} {
	return map[string]interface{}{
		"cookbook_name": name,
		"name":          fmt.Sprintf("%s-%s", name, version),
		"version":       version,
		"chef_type":     "cookbook_version",
		"json_class":    "Chef::CookbookVersion",
		"metadata": map[string]interface{}{
			"version":           version,
			"name":              name,
			"maintainer":        "test",
			"maintainer_email":  "test@example.com",
			"description":       "test cookbook",
			"long_description":  "",
			"license":           "Apache-2.0",
			"dependencies":      map[string]interface{}{},
		},
		"definitions": []interface{}{},
		"libraries":   []interface{}{},
		"attributes":  []interface{}{},
		"providers":   []interface{}{},
		"resources":   []interface{}{},
		"templates":   []interface{}{},
		"root_files":  []interface{}{},
		"files":       []interface{}{},
		"recipes":     []interface{}{},
		"frozen?":     false,
	}
}

func makeRecipeDiv(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"path":        "recipes/" + name,
		"checksum":    "",
		"specificity": "default",
	}
}

func checksum(data []byte) string {
	h := md5.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func makeReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(data))
}

func addRecipeFile(t *testing.T, org *organization.Organization, name string) string {
	data := []byte(name)
	chksum := checksum(data)
	fs, err := filestore.New(org, chksum, makeReadCloser(data), int64(len(data)))
	if err != nil {
		t.Fatalf("failed to create filestore item: %s", err.Error())
	}
	fs.Save()
	return chksum
}



func TestCookbookNew(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, err := New(org, "test-cookbook")
	if err != nil {
		t.Fatalf("New() failed: %s", err.Error())
	}
	if cb.Name != "test-cookbook" {
		t.Errorf("expected name test-cookbook, got %s", cb.Name)
	}
	if cb.NumVersions() != 0 {
		t.Errorf("expected 0 versions, got %d", cb.NumVersions())
	}
}

func TestCookbookNewInvalidName(t *testing.T) {
	org := setupCookbookTestOrg(t)
	_, err := New(org, "bad name")
	if err == nil {
		t.Fatal("expected error for invalid cookbook name")
	}
}

func TestCookbookSaveAndGet(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "save-cookbook")
	if err := cb.Save(); err != nil {
		t.Fatalf("Save() failed: %s", err.Error())
	}
	cb2, err := Get(org, "save-cookbook")
	if err != nil {
		t.Fatalf("Get() failed: %s", err.Error())
	}
	if cb2.Name != "save-cookbook" {
		t.Errorf("expected save-cookbook, got %s", cb2.Name)
	}
}

func TestCookbookGetNotFound(t *testing.T) {
	org := setupCookbookTestOrg(t)
	_, err := Get(org, "missing")
	if err == nil {
		t.Fatal("expected error for missing cookbook")
	}
}

func TestCookbookDoesExist(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "exists-cookbook")
	cb.Save()
	found, err := DoesExist(org, "exists-cookbook")
	if err != nil {
		t.Fatalf("DoesExist() failed: %s", err.Error())
	}
	if !found {
		t.Error("expected cookbook to exist")
	}
	found, _ = DoesExist(org, "missing-cookbook")
	if found {
		t.Error("expected cookbook not to exist")
	}
}

func TestCookbookDelete(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "delete-cookbook")
	cb.Save()
	if err := cb.Delete(); err != nil {
		t.Fatalf("Delete() failed: %s", err.Error())
	}
	_, err := Get(org, "delete-cookbook")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestCookbookGetList(t *testing.T) {
	org := setupCookbookTestOrg(t)
	for _, name := range []string{"a", "b", "c"} {
		cb, _ := New(org, name)
		cb.Save()
	}
	list := GetList(org)
	if len(list) != 3 {
		t.Errorf("expected 3 cookbooks, got %d", len(list))
	}
}

func TestCookbookAllCookbooks(t *testing.T) {
	org := setupCookbookTestOrg(t)
	for _, name := range []string{"all-a", "all-b"} {
		cb, _ := New(org, name)
		cb.Save()
	}
	all := AllCookbooks(org)
	if len(all) != 2 {
		t.Errorf("expected 2 cookbooks, got %d", len(all))
	}
}

func TestCookbookInfoHash(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "info-cookbook")
	cbv, _ := cb.NewVersion("1.0.0", makeVersionJSON("info-cookbook", "1.0.0"))
	_ = cbv
	info := cb.InfoHash("all")
	if info["url"] == "" {
		t.Error("expected non-empty URL")
	}
	if _, ok := info["versions"]; !ok {
		t.Error("expected versions key")
	}
}

func TestCookbookLatestAndConstrained(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "latest-cookbook")
	for _, ver := range []string{"1.0.0", "2.0.0"} {
		cbv, gerr := cb.NewVersion(ver, makeVersionJSON("latest-cookbook", ver))
		if gerr != nil {
			t.Fatalf("NewVersion(%s) failed: %s", ver, gerr.Error())
		}
		cb.Versions[ver] = cbv
	}
	cb.UpdateLatestVersion()
	latest := cb.LatestVersion()
	if latest.Version != "2.0.0" {
		t.Errorf("expected latest 2.0.0, got %s", latest.Version)
	}
	constrained := cb.LatestConstrained("= 1.0.0")
	if constrained == nil || constrained.Version != "1.0.0" {
		t.Errorf("expected constrained 1.0.0, got %v", constrained)
	}
}

func TestCookbookURLType(t *testing.T) {
	cb := &Cookbook{Name: "url-cookbook"}
	if cb.URLType() != "cookbooks" {
		t.Errorf("expected url type cookbooks, got %s", cb.URLType())
	}
}

func TestCookbookVersionGetName(t *testing.T) {
	cbv := &CookbookVersion{Name: "cbv-1.0.0"}
	if cbv.GetName() != "cbv-1.0.0" {
		t.Errorf("expected cbv-1.0.0, got %s", cbv.GetName())
	}
}

func TestCookbookRecipeList(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "recipe-cookbook")
	json := makeVersionJSON("recipe-cookbook", "1.0.0")
	cbv, err := cb.NewVersion("1.0.0", json)
	if err != nil {
		t.Fatalf("NewVersion() failed: %s", err.Error())
	}
	cbv.Recipes = []map[string]interface{}{{"name": "default.rb"}}
	recipes, gerr := cbv.RecipeList()
	if gerr != nil {
		t.Fatalf("RecipeList() failed: %s", gerr.Error())
	}
	if len(recipes) != 1 || recipes[0] != "recipe-cookbook" {
		t.Errorf("expected [recipe-cookbook], got %v", recipes)
	}
}

func TestCookbookVersionToJSON(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "json-cookbook")
	cb.Save()
	json := makeVersionJSON("json-cookbook", "1.0.0")
	cbv, err := cb.NewVersion("1.0.0", json)
	if err != nil {
		t.Fatalf("NewVersion() failed: %s", err.Error())
	}
	j := cbv.ToJSON("GET")
	if j["version"] != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %v", j["version"])
	}
}

func TestCookbookNumVersions(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "num-cookbook")
	if cb.NumVersions() != 0 {
		t.Errorf("expected 0 versions initially, got %d", cb.NumVersions())
	}
}

func TestCookbookNewVersion(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "newver-cookbook")
	cb.Save()
	cbv, err := cb.NewVersion("1.0.0", makeVersionJSON("newver-cookbook", "1.0.0"))
	if err != nil {
		t.Fatalf("NewVersion() failed: %s", err.Error())
	}
	if cbv.CookbookName != "newver-cookbook" {
		t.Errorf("expected newver-cookbook, got %s", cbv.CookbookName)
	}
}

func TestCookbookContainerType(t *testing.T) {
	cb := &Cookbook{Name: "ct-cookbook"}
	if cb.ContainerType() != "cookbooks" {
		t.Errorf("expected container type cookbooks, got %s", cb.ContainerType())
	}
	if cb.ContainerKind() != "containers" {
		t.Errorf("expected container kind containers, got %s", cb.ContainerKind())
	}
}

func TestCookbookDeleteVersion(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "delver-cookbook")
	cby, _ := cb.NewVersion("1.0.0", makeVersionJSON("delver-cookbook", "1.0.0"))
	_ = cby
	if cb.NumVersions() != 1 {
		t.Fatalf("expected 1 version, got %d", cb.NumVersions())
	}
	if err := cb.DeleteVersion("1.0.0"); err != nil {
		t.Fatalf("DeleteVersion() failed: %s", err.Error())
	}
	if cb.NumVersions() != 0 {
		t.Errorf("expected 0 versions after delete, got %d", cb.NumVersions())
	}
	_, err := cb.GetVersion("1.0.0")
	if err == nil {
		t.Fatal("expected error getting deleted version")
	}
}

func TestCookbookDoesVersionExist(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "dvexist-cookbook")
	cbv, _ := cb.NewVersion("1.0.0", makeVersionJSON("dvexist-cookbook", "1.0.0"))
	_ = cbv
	found, err := cb.DoesVersionExist(org, "1.0.0")
	if err != nil {
		t.Fatalf("DoesVersionExist() failed: %s", err.Error())
	}
	if !found {
		t.Error("expected version to exist")
	}
	found = false
	found, _ = cb.DoesVersionExist(org, "9.9.9")
	if found {
		t.Error("expected version not to exist")
	}
}

func TestCookbookGetVersion(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "getver-cookbook")
	cbv, _ := cb.NewVersion("1.0.0", makeVersionJSON("getver-cookbook", "1.0.0"))
	_ = cbv
	got, err := cb.GetVersion("1.0.0")
	if err != nil {
		t.Fatalf("GetVersion() failed: %s", err.Error())
	}
	if got.Version != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", got.Version)
	}
}

func TestCookbookLatestVersion(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "latestver-cookbook")
	for _, ver := range []string{"1.0.0", "2.0.0"} {
		cb.NewVersion(ver, makeVersionJSON("latestver-cookbook", ver))
	}
	latest := cb.LatestVersion()
	if latest == nil || latest.Version != "2.0.0" {
		t.Errorf("expected latest 2.0.0, got %v", latest)
	}
	latest, _ = cb.GetVersion("_latest")
	if latest == nil || latest.Version != "2.0.0" {
		t.Errorf("expected _latest 2.0.0, got %v", latest)
	}
}

func TestCookbookConstrainedInfoHash(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "constrained-cookbook")
	for _, ver := range []string{"1.0.0", "2.0.0"} {
		cb.NewVersion(ver, makeVersionJSON("constrained-cookbook", ver))
	}
	info := cb.ConstrainedInfoHash("all", ">= 2.0.0")
	versions := info["versions"].([]interface{})
	if len(versions) != 1 {
		t.Errorf("expected 1 version matching constraint, got %d", len(versions))
	}
}

func TestCookbookCookbookRecipes(t *testing.T) {
	org := setupCookbookTestOrg(t)
	chksum := addRecipeFile(t, org, "default.rb")
	cb, _ := New(org, "cbrecipes-cookbook")
	json := makeVersionJSON("cbrecipes-cookbook", "1.0.0")
	json["recipes"] = []interface{}{
		map[string]interface{}{
			"name":        "default.rb",
			"path":        "recipes/default.rb",
			"checksum":    chksum,
			"specificity": "default",
		},
	}
	cb.NewVersion("1.0.0", json)
	rlist, err := CookbookRecipes(org)
	if err != nil {
		t.Fatalf("CookbookRecipes() failed: %s", err.Error())
	}
	if len(rlist) != 1 || rlist[0] != "cbrecipes-cookbook" {
		t.Errorf("expected [cbrecipes-cookbook], got %v", rlist)
	}
}

func TestCookbookUniverse(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "universe-cookbook")
	cb.NewVersion("1.0.0", makeVersionJSON("universe-cookbook", "1.0.0"))
	u := Universe(org)
	if _, ok := u["universe-cookbook"]; !ok {
		t.Error("expected universe-cookbook in universe")
	}
}

func TestCookbookCookbookLatest(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "cblatest-cookbook")
	cb.NewVersion("1.0.0", makeVersionJSON("cblatest-cookbook", "1.0.0"))
	latest := CookbookLatest(org)
	if _, ok := latest["cblatest-cookbook"]; !ok {
		t.Error("expected cblatest-cookbook in latest")
	}
}

func TestCookbookUpdateVersion(t *testing.T) {
	org := setupCookbookTestOrg(t)
	cb, _ := New(org, "update-cookbook")
	cbv, _ := cb.NewVersion("1.0.0", makeVersionJSON("update-cookbook", "1.0.0"))
	json := makeVersionJSON("update-cookbook", "1.0.0")
	json["metadata"] = map[string]interface{}{
		"version":          "1.0.0",
		"name":             "update-cookbook",
		"maintainer":       "updated",
		"maintainer_email": "updated@example.com",
		"description":      "updated cookbook",
		"long_description": "",
		"license":          "MIT",
	}
	if err := cbv.UpdateVersion(json, ""); err != nil {
		t.Fatalf("UpdateVersion() failed: %s", err.Error())
	}
	if cbv.Metadata["maintainer"] != "updated" {
		t.Errorf("expected maintainer updated, got %v", cbv.Metadata["maintainer"])
	}
}
