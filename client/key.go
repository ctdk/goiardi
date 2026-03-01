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
	"encoding/json"
	"github.com/ctdk/chefcrypto"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/gerror"
	"github.com/ctdk/goiardi/logger"
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
	clientId int64
	id int64
}

// KeyInfo represents a key more sparsely for listing.
type KeyInfo struct {
	Name string `json:"name"`
	Uri string `json:"uri"`
	Expired bool `json:"expired"`
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
		k, err := c.getKeySQL(name)
		if err != nil {
			logger.Errorf("Error getting client key from DB: %s", err.Error())
		}
		return k
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
		keys, err := c.getAllKeysSQL()
		if err != nil {
			logger.Errorf("Error getting allclient keys from DB: %s", err.Error())
		}
		return keys
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

// GetKeyInfo returns an array of KeyInfo objects representing the shorter
// version of a key's information for one list endpoint.
func (c *Client) GetKeyInfo() []*KeyInfo {
	keys := c.GetKeysArray()
	keyInfo := make([]*KeyInfo, len(keys))
	now := time.Now()
	for i, v := range keys {
		ki := new(KeyInfo)
		ki.Name = v.Name
		ki.Expired = now.After(v.ExpirationDate)
		ki.Uri = util.CustomObjURL(c, util.JoinStr("/keys/", ki.Name))
		keyInfo[i] = ki
	}

	return keyInfo
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

// GenerateDefaultKeys is a convenience wrapper around GenerateNamedKeys that
// generates a default set of public and private keys without the caller needing
// to specify the name or expiration.
func (c *Client) GenerateDefaultKeys() (string, util.Gerror) {
	priv, _, err := c.GenerateNamedKeys("default", infinity.Infinity)
	if err != nil {
		return "", err
	}
	return priv, nil
}

// SetNamedKey adds the Key object to the client's list of keys.
func (c *Client) SetNamedKey(k *Key) util.Gerror {
	if config.UsingExternalSecrets() {
		// do whatever, not implemented yet
	}

	// if using the DB, but not external secrets
	if config.UsingDB() {
		return c.saveKeyPostgreSQL(k)
	}

	keys := c.GetAllPublicKeys()
	keys[k.Name] = k
	return c.saveAllKeys(keys)
}

// DeleteKey deletes a public key.
func (c *Client) DeleteKey(name string) util.Gerror {
	if config.UsingExternalSecrets() {
		// do whatever, not implemented yet
	}

	// if using the DB, but not external secrets
	if config.UsingDB() {
		return c.deleteKeySQL(name)
	}

	keys := c.GetAllPublicKeys()
	delete(keys, name)
	return c.saveAllKeys(keys)
}

// DeleteAllKeys deletes all of a client's public keys.
func (c *Client) DeleteAllKeys() util.Gerror {
	if config.UsingExternalSecrets() {
		// do whatever, not implemented yet
	}

	// if using the DB, but not external secrets
	if config.UsingDB() {
		return c.deleteAllKeysSQL()
	}

	ds := datastore.New()
	ds.Delete(c.org.DataKey("client-keys"), c.Name)
	return nil
}

// Only useful for in-mem store, I think
func (c *Client) saveAllKeys(keys map[string]*Key) util.Gerror {
	ds := datastore.New()
	ds.Set(c.org.DataKey("client-keys"), c.Name, keys)
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

// MarshalJSON and UnmarshalJSON methods for Keys need to be provided because
// the expiration date field needs special handing in case it's the common case
// where the expiration date is "infinity".
func (k *Key) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["name"] = k.Name
	m["public_key"] = k.PublicKey
	if k.ExpirationDate.Equal(infinity.Infinity) {
		m["expiration_date"] = "infinity"
	} else {
		m["expiration_date"] = k.ExpirationDate
	}
	return json.Marshal(m)
}

// UnmarshalJSON, fortunately, doesn't need anything special.
func (k *Key) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &k)
}
