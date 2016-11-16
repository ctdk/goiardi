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

// Package secret contains functions for handling secrets, like public/private
// keys stored outside of goiardi.
package secret

import (
	"github.com/ctdk/goiardi/util"
)

type ActorKeyer interface {
	PublicKey() string
	SetPublicKey(interface{}) error
	util.GoiardiObj
}

type secretSource interface {
	getPublicKey(ActorKeyer) (string, error)
	setPublicKey(ActorKeyer, string) error
}

var secretStore secretSource

func ConfigureSecretStore() error {
	// will be a switch here for the type of secret backend
	var err error
	secretStore, err = configureVault()
	if err != nil {
		return err
	}
	return nil
}

func GetPublicKey(c ActorKeyer) (string, error) {
	return secretStore.getPublicKey(c)
}

func SetPublicKey(c ActorKeyer, pubKey string) error {
	return secretStore.setPublicKey(c, pubKey)
}
