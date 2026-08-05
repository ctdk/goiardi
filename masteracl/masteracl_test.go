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

package masteracl

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ctdk/goiardi/config"
)

type fakeMasterActor struct {
	name     string
	isUser   bool
	isClient bool
	id       int64
}

func (a fakeMasterActor) GetName() string { return a.name }
func (a fakeMasterActor) IsUser() bool    { return a.isUser }
func (a fakeMasterActor) IsClient() bool { return a.isClient }
func (a fakeMasterActor) GetId() int64   { return a.id }

func setupMasterACLTest(t *testing.T) func() {
	t.Helper()
	tmpDir, err := ioutil.TempDir("", "masteracl-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err.Error())
	}
	config.Config = &config.Conf{}
	config.Config.PolicyRoot = tmpDir
	return func() {
		os.RemoveAll(tmpDir)
	}
}

func TestMasterCheckPermClientDenied(t *testing.T) {
	cleanup := setupMasterACLTest(t)
	defer cleanup()

	client := fakeMasterActor{name: "some-client", isClient: true, isUser: false}
	ok, err := MasterCheckPerm(client, Organizations, "read")
	if err == nil {
		t.Error("expected error for client actor")
	}
	if ok {
		t.Error("expected client to be denied")
	}
}

func TestMasterCheckPermAdminAllowed(t *testing.T) {
	cleanup := setupMasterACLTest(t)
	defer cleanup()

	admin := fakeMasterActor{name: "pivotal", isUser: true, isClient: false}
	ok, err := MasterCheckPerm(admin, Organizations, "read")
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if !ok {
		t.Error("expected pivotal to be allowed organizations read")
	}
}

func TestMasterCheckPermNonAdminDenied(t *testing.T) {
	cleanup := setupMasterACLTest(t)
	defer cleanup()

	user := fakeMasterActor{name: "nobody", isUser: true, isClient: false}
	ok, err := MasterCheckPerm(user, Organizations, "read")
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if ok {
		t.Error("expected non-admin user to be denied")
	}
}

func TestMasterCheckPermReindexAdminAllowed(t *testing.T) {
	cleanup := setupMasterACLTest(t)
	defer cleanup()

	admin := fakeMasterActor{name: "pivotal", isUser: true, isClient: false}
	ok, err := MasterCheckPerm(admin, Reindex, "create")
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if !ok {
		t.Error("expected pivotal to be allowed reindex create")
	}
}
