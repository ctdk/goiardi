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

// Functions for using hashicorp vault (https://www.vaultproject.io/) to store
// secrets in goiardi.

import(
	"fmt"
	vault "github.com/hashicorp/vault/api"
)

// make this a pool later?

type vaultSecretStore struct {
	*vault.Client
}

func configureVault() (*vaultSecretStore, error) {
	// use the VAULT_* environment variables to configure vault access,
	// at least for now
	conf := vault.DefaultConfig()
	if err := conf.ReadEnvironment(); err != nil {
		return nil, err
	}
	c, err := vault.NewClient(conf)
	if err != nil {
		return nil, err
	}

	v := &vaultSecretStore{ c }
	return v, nil
}

func (v *vaultSecretStore) getPublicKey(c ActorKeyer) (string, error) {
	path := makePath(c)
	s, err := v.Logical().Read(path)
	if err != nil {
		err := fmt.Errorf("Failed to read %s from vault: %s", path, err)
		return "", err
	}
	pk := s[Data]["pubKey"]
	if pk == nil {
		err := fmt.Errorf("no data for %s from vault", path)
		return "", err
	}
	var pubKey string
	switch pk.(type) {
	case string:
		pubKey = pk.(string)
	default:
		err := fmt.Errorf("pubKey from vault was not a string, but a %T!", pk)
		return "", err
	}
	return pubKey, nil
}

func (v *vaultSecretStore) setPublicKey(c ActorKeyer, pubKey string) error {
	path := makePath(c)
	_, err := v.Logical().Write(path, map[string]interface{}{
		"pubKey": pubKey,
	})
	return err
}

func makePath(c ActorKeyer) string {
	return fmt.Sprintf("keys/%s/%s", c.URLType(), c.GetName())
}
