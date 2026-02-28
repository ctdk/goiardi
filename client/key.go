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

package client

import (
	"github.com/ctdk/chefcrypto"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/gerror"
	"github.com/ctdk/goiardi/infinity"
	"github.com/ctdk/goiardi/util"
	"time"
)

// WIP

// TODO: handle keys stored in an external secret store

// Key represents a client public key. Chef clients can now have more than one
// set of keys, so these have moved to their own table and type.
type Key struct {
	PublicKey string `json:"public_key"`
	Name string `json:"name"`
	// The expiration_date can be "infinity", which requires a bit of hoop
	// jumping on the golang end.
	ExpirationDate time.Time `json:"expiration_date"`
	client_id int64
	id int64
}

// DefaultPublicKey is a convenience method that returns the client's default
// public key without having to provide the name.
func (c *Client) DefaultPublicKey() *Key {
	return c.NamedPublicKey("default")
}

// GetNamedPublicKey gets a client's public key with the given name.
func (c *Client) NamedPublicKey(name string) *Key {
	if config.UsingExternalSecrets() {
		// do whatever, not implemented yet
	}

	// if using the DB, but not external secrets
	if config.UsingDB() {

	}

	k := c.GetAllPublicKeys()
	return k[name]
}

// GetAllPublicKeys returns a map of all of the client's public keys.
func (c *Client) GetAllPublicKeys() map[string]*Key {
	if config.UsingExternalSecrets() {
		// do whatever, not implemented yet
	}

	// if using the DB, but not external secrets
	if config.UsingDB() {

	}

	ds := datastore.New()
	var keys map[string]*Key
	if ktmp, found := ds.Get(c.org.DataKey("client-keys"), c.Name); !found {
		keys = make(map[string]*Key)
	} else {
		keys = ktmp.(map[string]*Key)
	}

	return keys
}

// GetKeysArray returns an array of a client's public keys.
func (c *Client) GetKeysArray() []*Key {
	keymap := c.GetAllPublicKeys()
	keys := make([]*Key, len(keymap))
	i := 0
	for _, v := range keymap {
		keys[i] = v
		i++
	}
	return keys
}

// GenerateNamedKeys generates a new set of RSA keys for a client and stored
// with the provided name.
func (c *Client) GenerateNamedKeys(name string, expiration time.Time) (string, *Key, util.Gerror) {
	privPem, pubPem, err := chefcrypto.GenerateRSAKeys()
	if err != nil {
		return "", nil, util.CastErr(err)
	}

	if expiration.IsZero() {
		expiration = infinity.Infinity
	}

	var nerr util.Gerror
	if name, nerr = util.ValidateAsString(name); nerr != nil {
		return "", nil, nerr
	}

	k := &Key{Name: name, PublicKey: pubPem, ExpirationDate: expiration}

	// and (try to) save the key
	if kerr := c.SetNamedKey(k); kerr != nil {
		return "", nil, kerr
	}

	return privPem, k, nil
}

// SetNamedKey adds the Key object to the client's list of keys.
func (c *Client) SetNamedKey(k *Key) util.Gerror {
	if config.UsingExternalSecrets() {
		// do whatever, not implemented yet
	}

	// if using the DB, but not external secrets
	if config.UsingDB() {

	}

	keys := c.GetAllPublicKeys()
	keys[k.Name] = k
	return c.saveAllKeys(keys)
}

// Only useful for in-mem store, I think
func (c *Client) saveAllKeys(keys map[string]*Key) util.Gerror {
	ds := datastore.New()
	ds.Set(c.org.DataKey("client-keys"), c.Name, &keys)
	return nil
}

// functions and methods directly working on the keys

// KeyFromJSON builds a key object from a provided JSON hash, checking for
// correctness and validity.
func KeyFromJSON(raw interface{}) (*Key, util.Gerror) {
	switch raw := raw.(type) {
	case map[string]interface{}:
		if _, nErr := util.ValidateAsString(raw["name"]); nErr != nil {
			return nil, nErr
		}
		if pkok, pkerr := ValidatePublicKey(raw["public_key"]); !pkok {
			return nil, pkerr
		}
		return newKey(raw)
	default:
		return nil, gerror.Errorf("Bad key JSON provided (T %T) (val '%v')", raw, raw)
	}
}

// after the key and name validations
func newKey(k map[string]interface{}) (*Key, util.Gerror) {
	nk := new(Key)
	nk.Name = k["name"].(string)
	nk.PublicKey = k["public_key"].(string)
	if xt, ok := k["expiration_date"].(string); !ok || xt == "infinity" {
		nk.ExpirationDate = infinity.Infinity
	} else {
		t, err := time.Parse(time.RFC3339, xt)
		if err != nil {
			return nil, util.CastErr(err)
		}
		nk.ExpirationDate = t
	}

	return nk, nil
}
