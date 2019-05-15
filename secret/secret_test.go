// +build !novault

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

package secret

import (
	"fmt"
	"github.com/ctdk/chefcrypto"
	vault "github.com/hashicorp/vault/api"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Only run these particular tests if vault is installed
var vaultInstalled bool

const (
	token     = "f1d77d43-0a27-f05a-5426-08bd20a6311d"
	vaultAddr = "127.0.0.1:28022"
	pubKey    = "ABCDEF123456"
)

var signingKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAorvBRY/x2nNvLW6m8odIdbOT2wR6mbC9QVApz+dPg4ZMI/+r
AJZeTPv4g4n4qzAyU+4UAIRuAzSM+nHzwAlan6T/rtZ52a8k9nKlQ7e0mXAiL2dj
bRcXCp+60NU1HlWd6GfN34fA8/8fMbRRQ9an0ikBFz9YJv3izM/QnshXM73cdGyh
zFa29PX3jOgAyN9k7MmpR8JDbPEaX4Shny2BbqLgtHhEwRa0GNlf6qTdv7sf+g0a
T2FllzHMWv6v/uLA2L1VZO5x6QDJQPT+Xl1EJNHpPrXgMHm3tU4nJZD9scQD6dsZ
a7URhpXI6ckRaiF+tIMXfLyrRgRJ5TnOynJJhwIDAQABAoIBAEMkdHnfCkq3lgeI
wBkQ+DSYA0k6b9s5sNxh1t6Q8Z2yq3eu5T84y2+4BrE/G/qFyD4Y3OfZvApWhFRQ
7+Er+tgjm9rnYx8NxJJqewWVpk4olfI5FizMehVIEixXy7LYYG6jZa30tQf3G0fG
vkDfMB7mDC0rVZPA7PLUS583ycLuyrCUu2A8HLuq0aKEyyjDs2rpivT9emveLvmM
B1HnyFpMub2OPz/tLxgR+ZAkR14qBgZOZq64OxVdFlkY9w1d8VcCG5QARbQF4IIC
TU2k5nRvSP7T+hvuiE8cjngX8N4D3Kgyb/wn3x5pYJLk10vIkJ+iq+upuyjLtToP
Z0klrgECgYEA0DQ52q6aElKkeSKQX+2ZZDFcW7Q8skeA+QvZke+Gh80h6WEbuSEH
Y4GjAMP5vBobfCYg0JljaRSK7VATiM5K7MkKwALu1qptuv2aMrfLvpstEuUkIPk7
W7rbVbDqQVBNm0RsG310XUkdtxpsGVNmfQZpGuxWDYb5FB1Vv/aOqikCgYEAyBdN
7k1hyMOSgQQIbTmn5QZJWMyzeISjE3gSDxmLddL9SthI4iMfiEkJ+QiyLguOVOds
A3OnjOSU4sVlZTyDPUrkst0J93RkNKoH8V6YTKUB3rL4hEqNFp+DzoiV/QGIB+Nl
rJmAXr8OCy4jT3yGoPVqxGGKTPvfmk73bz0yLC8CgYB9Zfcybtb9EildjCPIoyIv
5krqMLQd7FMRrMFt1AYC5Nn35jT8W5hHply2qVJQqKjFLXG2MaxeLbE/HWumihk2
ZB/FZf7T6/ILHZMx2OEt++g56SjJc1L8/J3+DoAItoUHbz5tkDH1vYPYNGHHHyQr
SSEkxhNOMmzyYHi1FZr3gQKBgQDEmfFetvXfmBp5XgcRm1cWt4iWEfw97MV3OcE0
yPq4uKlcQwvJ9ozjEjEUWrEIgR5G2mTNN3RoAakw8JfiUwT40n/IJ1vFor1a6b6I
MVQf6vndYajCA6aBlhaidp45TKnmZk7euqzha4RXA+x6C3cU7E8NynFjlxBrxC9n
Q4/qzQKBgHhLsgLobGlKoPfhZ3JyCiH7mvoDAiWyoC0dO5EkCh36YHappl+t71y+
tWdMV8eXdXrb06mkgko8TlfknM23pxVXIskVjV5+NpUeDhm6LDiliw0pHS0mctDv
cnY8AtHbM15v5i0pXYjtwC/qIBA7IJnIXXE29BKFSC0IcIeLPqVP
-----END RSA PRIVATE KEY-----`

var signingPath = "keys/shovey/signing"

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

	// try to connect to vault 3 times before giving up.
	numTries := 3
	for si := 0; si < numTries; si++ {
		time.Sleep(3)
		mount := exec.Command(vaultPath, "secrets", "enable", "-path=keys", "generic")
		mount.Env = append(mount.Env, fmt.Sprintf("VAULT_ADDR=http://%s", vaultAddr))
		mount.Env = append(mount.Env, fmt.Sprintf("VAULT_TOKEN=%s", token))
		err = mount.Run()
		if err == nil {
			break
		} else {
			log.Printf("try #%d to mount vault secrets failed, will try up to %d times.", si+1, numTries)
		}
	}

	if err != nil {
		log.Fatalf("Tried to mount vault secrets 3 times, but failed: '%s'", err.Error())
	}

	conf := vault.DefaultConfig()
	if err := conf.ReadEnvironment(); err != nil {
		log.Fatalf("error reading vault environment: %s", err.Error())
	}
	cl, err := vault.NewClient(conf)
	if err != nil {
		log.Fatalf("error setting up vault client: %s", err.Error())
	}
	_, err = cl.Logical().Write(signingPath, map[string]interface{}{
		"RSAKey": signingKey,
	})
	if err != nil {
		log.Fatalf("error writing signing key to vault: %s", err.Error())
	}

	c = &keyer{name: "foobar"}
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
	cs := []*keyer{&keyer{name: "bleek1"}, &keyer{name: "bleek2"}, &keyer{name: "bleek3"}}

	keys := []string{"12345", "abcdef", "eekamouse"}

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

func TestDeleteKey(t *testing.T) {
	k := &keyer{name: "key_keyerson"}
	key := "456123000"
	err := SetPublicKey(k, key)
	if err != nil {
		t.Error(err)
	}
	err = DeletePublicKey(k)
	if err != nil {
		t.Errorf("error deleting key: %s", err.Error())
	}
	pk, pkerr := GetPublicKey(k)
	if pk != "" || pkerr == nil {
		t.Errorf("public key '%s' was found after it was deleted", pk)
	}
}

func TestSetPasswdHash(t *testing.T) {
	k := &keyer{name: "bob_keyman"}
	pass := "foobarbaz"
	salt, err := chefcrypto.GenerateSalt()
	if err != nil {
		t.Error(err)
	}
	hash, err := chefcrypto.HashPasswd(pass, salt)
	if err != nil {
		t.Error(err)
	}
	err = SetPasswdHash(k, hash)
	if err != nil {
		t.Errorf("Problem setting passwd hash: %s", err.Error())
	}
}

func TestGetPasswdHash(t *testing.T) {
	k := &keyer{name: "jebediah_keyman"}
	pass := "foobarbaz"
	salt, err := chefcrypto.GenerateSalt()
	if err != nil {
		t.Error(err)
	}
	hash, err := chefcrypto.HashPasswd(pass, salt)
	if err != nil {
		t.Error(err)
	}
	err = SetPasswdHash(k, hash)
	if err != nil {
		t.Errorf("Problem setting passwd hash (in GetPasswdHash test): %s", err.Error())
	}

	h, err := GetPasswdHash(k)
	if err != nil {
		t.Errorf("Problem getting passwd hash: %s", err.Error())
	}
	if h != hash {
		t.Errorf("Password hashes did not match, expected %s, got %s", hash, h)
	}
}

func TestDeletePasswdHash(t *testing.T) {
	k := &keyer{name: "bill_keyman"}
	pass := "foobarbaz"
	salt, err := chefcrypto.GenerateSalt()
	if err != nil {
		t.Error(err)
	}
	hash, err := chefcrypto.HashPasswd(pass, salt)
	if err != nil {
		t.Error(err)
	}
	err = SetPasswdHash(k, hash)
	if err != nil {
		t.Errorf("Problem setting passwd hash (in DeletePasswdHash test): %s", err.Error())
	}

	err = DeletePasswdHash(k)
	if err != nil {
		t.Errorf("Problem deleting passwd hash (in DeletePasswdHash test): %s", err.Error())
	}

	h, err := GetPasswdHash(k)
	if err == nil {
		t.Errorf("No error fetching password hash")
	}
	if h != "" {
		t.Errorf("hash %s unexpectedly found in vault", h)
	}
}

func TestGetSigningKey(t *testing.T) {
	_, err := GetSigningKey(signingPath)
	if err != nil {
		t.Errorf("error getting signing key: %s", err.Error())
	}
}
