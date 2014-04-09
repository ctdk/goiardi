/*
 * Copyright (c) 2013-2014, Jeremy Bingham (<jbingham@gmail.com>)
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

// Users and clients ended up having to be split apart after all, once adding
// the SQL backing started falling into place. Users are very similar to
// clients, except that they are unique across the whole server and can log in 
// via the web interface, while clients are only unique across an organization
// and cannot log in over the web. Basically, users are generally for something
// you would do, while a client would be associated with a specific node.
//
// Note: At this time, organizations are not implemented, so the difference
// between clients and users is a little less stark.
package user

import (
	"github.com/ctdk/goiardi/data_store"
	"fmt"
	"github.com/ctdk/goiardi/chef_crypto"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/config"
	"net/http"
	"encoding/gob"
	"bytes"
)

type User struct {
	Username string `json:"username"`
	Name string `json:"name"`
	Email string `json:"email"`
	Admin bool `json:"admin"`
	pubKey string `json:"public_key"`
	passwd string
	Salt []byte
}


// Generate a new set of RSA keys for the user. The new private key is saved
// with the user object, the public key is given to the user and not saved on 
// the server at all. 
func (u *User) GenerateKeys() (string, error){
	priv_pem, pub_pem, err := chef_crypto.GenerateRSAKeys()
	if err != nil {
		return "", err
	}
	c.pubKey = pub_pem
	return priv_pem, nil
}

// Checks that the provided public key is valid. Wrapper around 
// chef_crypto.ValidatePublicKey(), but with a different error type.
func ValidatePublicKey(publicKey interface{}) (bool, util.Gerror) {
	ok, pkerr := chef_crypto.ValidatePublicKey(publicKey)
	var err util.Gerror
	if !ok {
		err = util.Errorf(pkerr.Error())
	}
	return ok, err
}

// Is the user an admin? If use-auth is false, this always returns true.
func (u *User) IsAdmin() bool {
	if !config.Config.UseAuth {
		return true
	}
	return u.Admin
}

// Users are never validators, so always return false. This is true even if
// auth mode is not on.
func (u *User) IsValidator() bool {
	return false
}

func (u *User) IsSelf(other *Actor) bool {
	if !config.Config.UseAuth {
		return true
	}
}

func (u *User) IsUser() bool {
	return true
}

func (u *User) IsClient() bool {
	return false
}

// Return the user's public key. Part of the Actor interface.
func (u *User) PublicKey() string {
	return u.pubKey
}

// Validate and set the user's password. Will not set a password for a client.
func (u *User) SetPasswd(password string) util.Gerror {
	if len(password) < 6 {
		err := util.Errorf("Password must have at least 6 characters")
		return err
	}
	/* If those validations pass, set the password */
	var perr error
	u.passwd, perr = chef_crypto.HashPasswd(password, u.Salt)
	if perr != nil {
		err := util.Errorf(perr.Error())
		return err
	}
	return nil
}

// Check the provided password to see if it matches the stored password hash.
func (u *User) CheckPasswd(password string) util.Gerror {
	h, perr := chef_crypto.HashPasswd(password, u.Salt) 
	if perr != nil {
		err := util.Errorf(perr.Error())
		return err
	}
	if u.passwd != h {
		err := util.Errorf("password did not match")
		return err
	}
	
	return nil
}

func validateUserName(name string) util.Gerror {
	if !util.ValidateUserName(name) {
		err := util.Errorf("Field 'name' invalid")
		return err
	}
	return nil
}

func (u *User) GetName() string {
	return u.Username
}

func (u *User) URLType() string {
	return "users"
}
