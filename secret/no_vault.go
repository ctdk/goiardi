// +build novault

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
	"crypto/rsa"
	"errors"
)

// This file exists solely for the case where packaging vault and all its
// dependencies is enough of an ordeal that it's better left out. To do this,
// add "-tags 'novault'" to the build command.

var errNoVault = errors.New("Tried to use secrets, but this version of goiardi was compiled without vault support.")

type vaultSecretStore struct {
}

func configureVault() (*vaultSecretStore, error) {
	return nil, errNoVault
}

func (v *vaultSecretStore) getPublicKey(c ActorKeyer) (string, error) {
	return "", errNoVault
}

func (v *vaultSecretStore) setPublicKey(c ActorKeyer, f string) error {
	return errNoVault
}

func (v *vaultSecretStore) deletePublicKey(c ActorKeyer) error {
	return errNoVault
}

func (v *vaultSecretStore) setPasswdHash(c ActorKeyer, f string) error {
	return errNoVault
}

func (v *vaultSecretStore) getPasswdHash(c ActorKeyer) (string, error) {
	return "", errNoVault
}

func (v *vaultSecretStore) deletePasswdHash(c ActorKeyer) error {
	return errNoVault
}

func (v *vaultSecretStore) getSigningKey(f string) (*rsa.PrivateKey, error) {
	return nil, errNoVault
}
