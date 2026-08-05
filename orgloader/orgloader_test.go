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

package orgloader

import (
	"encoding/gob"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/organization"
)

func init() {
	gob.Register(&organization.Organization{})
}

func setupOrgloaderTest(t *testing.T) func() {
	t.Helper()
	tmpDir, err := ioutil.TempDir("", "orgloader-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err.Error())
	}
	config.Config = &config.Conf{}
	config.Config.PolicyRoot = tmpDir
	indexer.Initialize(config.Config, &organization.Organization{Name: "default"})
	return func() {
		os.RemoveAll(tmpDir)
	}
}

func TestOrgloaderNew(t *testing.T) {
	cleanup := setupOrgloaderTest(t)
	defer cleanup()

	org, err := New("orgloader-new", "Orgloader New")
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if org.Name != "orgloader-new" {
		t.Errorf("expected orgloader-new, got %s", org.Name)
	}
	if err := org.Delete(); err != nil {
		t.Fatalf("cleanup delete failed: %s", err.Error())
	}
}

func TestOrgloaderGet(t *testing.T) {
	cleanup := setupOrgloaderTest(t)
	defer cleanup()

	org, err := New("orgloader-get", "Orgloader Get")
	if err != nil {
		t.Fatalf("setup error: %s", err.Error())
	}
	got, err := Get("orgloader-get")
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if got.Name != "orgloader-get" {
		t.Errorf("expected orgloader-get, got %s", got.Name)
	}
	if err := org.Delete(); err != nil {
		t.Fatalf("cleanup delete failed: %s", err.Error())
	}
}

func TestOrgloaderGetMissing(t *testing.T) {
	cleanup := setupOrgloaderTest(t)
	defer cleanup()

	if _, err := Get("no-such-org"); err == nil {
		t.Error("expected error for missing org")
	}
}

func TestOrgloaderAllOrganizations(t *testing.T) {
	cleanup := setupOrgloaderTest(t)
	defer cleanup()

	org1, err := New("orgloader-all-1", "One")
	if err != nil {
		t.Fatalf("setup error: %s", err.Error())
	}
	org2, err := New("orgloader-all-2", "Two")
	if err != nil {
		t.Fatalf("setup error: %s", err.Error())
	}
	orgs, oerr := AllOrganizations()
	if oerr != nil {
		t.Fatalf("unexpected error: %s", oerr.Error())
	}
	if len(orgs) != 2 {
		t.Errorf("expected 2 orgs, got %d", len(orgs))
	}
	if err := org1.Delete(); err != nil {
		t.Fatalf("cleanup delete failed: %s", err.Error())
	}
	if err := org2.Delete(); err != nil {
		t.Fatalf("cleanup delete failed: %s", err.Error())
	}
}
