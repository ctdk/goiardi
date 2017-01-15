// +build !novault

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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/ctdk/goiardi/config"
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
	secretType    string
	created       time.Time
	renewable     bool
	ttl           time.Duration
	expires       time.Time
	stale         bool
	staleTryAgain time.Time
	staleTime     time.Time
	value         interface{}
}

type secretConvert func(interface{}) (interface{}, error)

func configureVault() (*vaultSecretStore, error) {
	conf := vault.DefaultConfig()
	if err := conf.ReadEnvironment(); err != nil {
		return nil, err
	}
	if config.Config.VaultAddr != "" {
		conf.Address = config.Config.VaultAddr
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

func (v *vaultSecretStore) getSecret(path string, secretType string) (interface{}, error) {
	if v.secrets[path] == nil {
		logger.Debugf("secret (%s) for %s is nil, fetching from vault", secretType, path)
		s, err := v.getSecretPath(path, secretType)
		if err != nil {
			return "", err
		}
		v.secrets[path] = s
	} else {
		logger.Debugf("using cached secret for %s", path)
	}
	return v.secretValue(v.secrets[path])
}

func (v *vaultSecretStore) getSecretPath(path string, secretType string) (*secretVal, error) {
	t := time.Now()
	s, err := v.Logical().Read(path)
	if err != nil {
		err := fmt.Errorf("Failed to read %s (%s) from vault: %s", path, secretType, err.Error())
		return nil, err
	}
	if s == nil {
		err := fmt.Errorf("No secret returned from vault for %s (%s)", path, secretType)
		return nil, err
	}
	p := s.Data[secretType]
	if p == nil {
		err := fmt.Errorf("no data for %s (%s) from vault", path, secretType)
		return nil, err
	}
	p, err = convertors(secretType)(p)
	if err != nil {
		return nil, err
	}
	sVal := newSecretVal(path, secretType, p, t, s)
	return sVal, nil
}

func (v *vaultSecretStore) setSecret(path string, secretType string, value interface{}) error {
	logger.Debugf("setting public key for %s (%s)", path, secretType)
	t := time.Now()
	_, err := v.Logical().Write(path, map[string]interface{}{
		secretType: value,
	})
	if err != nil {
		return err
	}
	s, err := v.Logical().Read(path)
	if err != nil {
		return fmt.Errorf("Error re-reading secret from vault after setting: %s", err.Error())
	}
	sVal := newSecretVal(path, secretType, value, t, s)
	v.secrets[path] = sVal
	return nil
}

func (v *vaultSecretStore) deleteSecret(path string) error {
	delete(v.secrets, path)
	_, err := v.Logical().Delete(path)
	if err != nil {
		return err
	}
	return nil
}

func (v *vaultSecretStore) getPublicKey(c ActorKeyer) (string, error) {
	v.m.RLock()
	defer v.m.RUnlock()
	path := makePubKeyPath(c)
	s, err := v.getSecret(path, "pubKey")
	switch s := s.(type) {
	case string:
		return s, err
	case []byte:
		return string(s), err
	case nil:
		return "", err
	default:
		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		err := fmt.Errorf("The type was wrong fetching the public key from vault: %T -- error, if any: %s", s, errStr)
		return "", err
	}
}

func (v *vaultSecretStore) setPublicKey(c ActorKeyer, pubKey string) error {
	v.m.Lock()
	defer v.m.Unlock()
	path := makePubKeyPath(c)
	return v.setSecret(path, "pubKey", pubKey)
}

func (v *vaultSecretStore) deletePublicKey(c ActorKeyer) error {
	v.m.Lock()
	defer v.m.Unlock()
	path := makePubKeyPath(c)
	return v.deleteSecret(path)
}

func makePubKeyPath(c ActorKeyer) string {
	return fmt.Sprintf("keys/%s/%s", c.URLType(), c.GetName())
}

func makeHashPath(c ActorKeyer) string {
	// strictly speaking only users actually have passwords, but in case
	// something else ever comes up, make the path a little longer.
	return fmt.Sprintf("keys/passwd/%s/%s", c.URLType(), c.GetName())
}

func newSecretVal(path string, secretType string, value interface{}, t time.Time, s *vault.Secret) *secretVal {
	sVal := new(secretVal)
	sVal.path = path
	sVal.secretType = secretType
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
		s2, err := v.getSecretPath(s.path, s.secretType)
		if !s.stale {
			if err != nil {
				logger.Debugf("error trying to renew the secret for %s: %s -- marking as stale", s.path, err.Error())
				s.stale = true
				s.staleTime = time.Now().Add(MaxStaleAgeSeconds * time.Second)
				s.staleTryAgain = time.Now().Add(StaleTryAgainSeconds * time.Second)
			} else {
				logger.Debugf("successfully renewed secret for %s", s.path)
				s = s2
			}
		} else if time.Now().After(s.staleTime) {
			if err != nil {
				err := fmt.Errorf("Couldn't renew the secret for %s before %d seconds ran out, giving up", s.path, MaxStaleAgeSeconds)
				return nil, err
			}
			logger.Debugf("successfully renewed secret for %s beforegiving up due to staleness", s.path)
			s = s2
		} else if time.Now().After(s.staleTryAgain) {
			if err != nil {
				logger.Debugf("error trying to renew the secret for %s: %s -- will renew again in %d seconds", s.path, err.Error(), StaleTryAgainSeconds)
				s.staleTryAgain = time.Now().Add(StaleTryAgainSeconds)
			} else {
				logger.Debugf("successfully renewed secret after being stale")
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

// shovey signing key

func (v *vaultSecretStore) getSigningKey(path string) (*rsa.PrivateKey, error) {
	v.m.RLock()
	defer v.m.RUnlock()
	s, err := v.getSecret(path, "RSAKey")
	switch s := s.(type) {
	case *rsa.PrivateKey:
		return s, err
	default:
		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		kerr := fmt.Errorf("RSA private key for shovey was not returned. An object of type %T was. Error, if any: %s", s, errStr)
		return nil, kerr
	}
}

// user passwd hash methods

func (v *vaultSecretStore) setPasswdHash(c ActorKeyer, pwhash string) error {
	v.m.Lock()
	defer v.m.Unlock()
	path := makeHashPath(c)
	return v.setSecret(path, "passwd", pwhash)
}

func (v *vaultSecretStore) getPasswdHash(c ActorKeyer) (string, error) {
	v.m.RLock()
	defer v.m.RUnlock()
	path := makeHashPath(c)
	s, err := v.getSecret(path, "passwd")
	switch s := s.(type) {
	case string:
		return s, err
	case []byte:
		return string(s), err
	case nil:
		return "", err
	default:
		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		err := fmt.Errorf("The type was wrong fetching the passwd hash from vault: %T -- error, if any: %s", s, errStr)
		return "", err
	}
}

func (v *vaultSecretStore) deletePasswdHash(c ActorKeyer) error {
	v.m.Lock()
	defer v.m.Unlock()
	path := makeHashPath(c)
	return v.deleteSecret(path)
}

// funcs to process secrets after fetching them from vault

func secretPassThrough(i interface{}) (interface{}, error) {
	return i, nil
}

func secretRSAKey(i interface{}) (interface{}, error) {
	p, ok := i.(string)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key in string form")
	}
	pBlock, _ := pem.Decode([]byte(p))
	if pBlock == nil {
		return nil, fmt.Errorf("invalid block size for private key for shovey from vault")
	}
	pk, err := x509.ParsePKCS1PrivateKey(pBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

func convertors(secretType string) secretConvert {
	switch secretType {
	case "RSAKey":
		return secretRSAKey
	default:
		return secretPassThrough
	}
}
