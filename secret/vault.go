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

import (
	"fmt"
	vault "github.com/hashicorp/vault/api"
	"github.com/tideland/golib/logger"
	"sync"
	"time"
)

// make this a pool later?

type vaultSecretStore struct {
	m       sync.RWMutex
	secrets map[string]*secretVal
	*vault.Client
}

const MaxStaleAgeSeconds = 3600 // configurable later, but make it an hour for
// now
const StaleTryAgainSeconds = 60 // try stale values again in a minute

type secretVal struct {
	path          string
	created       time.Time
	renewable     bool
	ttl           time.Duration
	expires       time.Time
	stale         bool
	staleTryAgain time.Time
	staleTime     time.Time
	value         interface{}
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

	var m sync.RWMutex
	secrets := make(map[string]*secretVal)
	v := &vaultSecretStore{m, secrets, c}
	return v, nil
}

func (v *vaultSecretStore) getPublicKey(c ActorKeyer) (string, error) {
	v.m.RLock()
	defer v.m.RUnlock()
	path := makePath(c)
	if v.secrets[path] == nil {
		logger.Debugf("secret for %s is nil, fetching from vault", path)
		s, err := v.getPublicKeySecretPath(path)
		if err != nil {
			return "", err
		}
		v.secrets[path] = s
	} else {
		logger.Debugf("using cached secret for %s", path)
	}
	return v.valueStr(v.secrets[path])
}

func (v *vaultSecretStore) getPublicKeySecretPath(path string) (*secretVal, error) {
	t := time.Now()
	s, err := v.Logical().Read(path)
	if err != nil {
		err := fmt.Errorf("Failed to read %s from vault: %s", path, err)
		return nil, err
	}
	pk := s.Data["pubKey"]
	if pk == nil {
		err := fmt.Errorf("no data for %s from vault", path)
		return nil, err
	}
	sVal := newSecretVal(path, pk, t, s)
	return sVal, nil
}

func (v *vaultSecretStore) setPublicKey(c ActorKeyer, pubKey string) error {
	v.m.Lock()
	defer v.m.Unlock()
	path := makePath(c)
	t := time.Now()
	_, err := v.Logical().Write(path, map[string]interface{}{
		"pubKey": pubKey,
	})
	if err != nil {
		return err
	}
	s, err := v.Logical().Read(path)
	if err != nil {
		return fmt.Errorf("Error re-reading secret from vault after setting: %s", err.Error())
	}
	sVal := newSecretVal(path, pubKey, t, s)
	v.secrets[path] = sVal
	return nil
}

func makePath(c ActorKeyer) string {
	return fmt.Sprintf("keys/%s/%s", c.URLType(), c.GetName())
}

func newSecretVal(path string, value interface{}, t time.Time, s *vault.Secret) *secretVal {
	sVal := new(secretVal)
	sVal.path = path
	sVal.created = t
	sVal.renewable = s.Renewable
	sVal.ttl = time.Duration(s.LeaseDuration) * time.Second
	sVal.expires = t.Add(sVal.ttl)
	sVal.value = value
	return sVal
}

func (s *secretVal) isExpired() bool {
	if s.ttl == 0 {
		return false
	}
	return time.Now().After(s.expires)
}

func (v *vaultSecretStore) secretValue(s *secretVal) (interface{}, error) {
	if s.isExpired() {
		logger.Debugf("trying to renew secret for %s", s.path)
		if !s.stale {
			s2, err := v.getPublicKeySecretPath(s.path)
			if err != nil {
				logger.Debugf("error trying to renew the secret for %s: %s -- marking as stale", s.path, err.Error())
				s.stale = true
				s.staleTime = time.Now().Add(MaxStaleAgeSeconds * time.Second)
				s.staleTryAgain = time.Now().Add(StaleTryAgainSeconds * time.Second)
			} else {
				s = s2
			}
		} else if time.Now().After(s.staleTime) {
			s2, err := v.getPublicKeySecretPath(s.path)
			if err != nil {
				err := fmt.Errorf("Couldn't renew the secret for %s before %d seconds ran out, giving up", s.path, MaxStaleAgeSeconds)
				return nil, err
			}
			s = s2
		} else if time.Now().After(s.staleTryAgain) {
			s2, err := v.getPublicKeySecretPath(s.path)
			if err != nil {
				logger.Debugf("error trying to renew the secret for %s: %s -- will renew again in %d seconds", s.path, err.Error(), StaleTryAgainSeconds)
				s.staleTryAgain = time.Now().Add(StaleTryAgainSeconds)
			} else {
				s = s2
			}
		}
	}
	return s.value, nil
}

func (v *vaultSecretStore) valueStr(s *secretVal) (string, error) {
	val, err := v.secretValue(s)
	if err != nil {
		return "", err
	}
	valStr, ok := val.(string)
	if !ok {
		err := fmt.Errorf("value for %s was not a string, but a %T!", s.path, val)
		return "", err
	}
	return valStr, nil
}
