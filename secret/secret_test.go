/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package secret

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Only run these particular tests if vault is installed
var vaultInstalled bool

const (
	token = "f1d77d43-0a27-f05a-5426-08bd20a6311d"
	vaultAddr = "127.0.0.1:28022"
	pubKey = "ABCDEF123456"
)

var c *keyer

type keyer struct {
	name string
}

func (k *keyer) GetName() string {
	return k.name
}

func (k *keyer) URLType() string {
	return "keyer"
}

func (k *keyer) PublicKey() string {
	return ""
}

func (k *keyer) SetPublicKey(i interface{}) error {
	return nil
}

func TestMain(m *testing.M) {
	vaultPath, err := exec.LookPath("vault")
	if err == nil {
		vaultInstalled = true
	} else {
		log.Printf("Vault is not installed, not running vault tests")
		return
	}
	cmd := exec.Command(vaultPath, "server", "-dev", fmt.Sprintf("-dev-listen-address=%s", vaultAddr), fmt.Sprintf("-dev-root-token-id=%s", token))

	e := cmd.Start()
	
	if e != nil {
		log.Fatalf("Err running vault: %s", e.Error())
	}

	defer func() {
		if err := recover(); err != nil {
		    cmd.Process.Kill()
		}
	}()

	os.Setenv("VAULT_ADDR", fmt.Sprintf("http://%s", vaultAddr))
	os.Setenv("VAULT_TOKEN", token)

	time.Sleep(3)
	mount := exec.Command(vaultPath, "mount", "-path=keys", "generic")
	mount.Env = append(mount.Env, fmt.Sprintf("VAULT_ADDR=http://%s", vaultAddr))
	mount.Env = append(mount.Env, fmt.Sprintf("VAULT_TOKEN=%s", token))
	err = mount.Run()
	if err != nil {
		log.Fatalf("Err mounting path in vault: '%s'", err.Error())
	}

	c = &keyer{ name: "foobar" }
	i := m.Run()
	cmd.Process.Kill()

	os.Exit(i)
}

func TestInit(t *testing.T) {
	if !vaultInstalled {
		return
	}
	// right now this only creates a vault backed service
	err := ConfigureSecretStore()
	if err != nil {
		t.Errorf("Error configuring store: %s", err.Error())
	}
}

func TestSetPublicKey(t *testing.T) {
	if !vaultInstalled {
		return
	}
	err := SetPublicKey(c, pubKey)
	if err != nil {
		t.Errorf("Error setting public key: %s", err.Error())
	}
}

func TestGetPublicKey(t *testing.T) {
	if !vaultInstalled {
		return
	}
	pk, err := GetPublicKey(c)
	if err != nil {
		t.Errorf("Error getting public key: %s", err.Error())
	}
	if pk != pubKey {
		t.Errorf("public key was incorrect: should have been '%s', got '%s'", pubKey, pk)
	}
}

func TestResetPublicKey(t *testing.T) {
	newKey := "herbaderbaderb"
	pk, err := GetPublicKey(c)
	if pk != pubKey {
		t.Errorf("public key was incorrect: should have been '%s', got '%s'", pubKey, pk)
	}
	err = SetPublicKey(c, newKey)
	if err != nil {
		t.Error(err)
	}
	pk2, err := GetPublicKey(c)
	if pk2 != newKey {
		t.Errorf("new public key was incorrect: should have been '%s', got '%s'", newKey, pk2)
	}
}

func TestMultipleObjKeys(t *testing.T) {
	cs := []*keyer{ &keyer{ name: "bleek1" }, &keyer{ name: "bleek2" }, &keyer{ name: "bleek3" } }
	
	keys := []string{ "12345", "abcdef", "eekamouse" }

	for i := 0; i < 3; i++ {
		err := SetPublicKey(cs[i], keys[i])
		if err != nil {
			t.Errorf("err setting key for client %d: %s", i, err.Error())
		}
	}
	for i := 0; i < 3; i++ {
		pk, err := GetPublicKey(cs[i])
		if err != nil {
			t.Errorf("err getting key for client %d: %s", i, err.Error())
		}
		if pk != keys[i] {
			t.Errorf("key for client %d did not match, expected '%s', but got '%s'", i, keys[i], pk)
		}
	}
}
