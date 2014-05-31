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
	"database/sql"
)

type User struct {
	Username string `json:"username"`
	Name string `json:"name"`
	Email string `json:"email"`
	Admin bool `json:"admin"`
	pubKey string `json:"public_key"`
	passwd string
	salt []byte
}

type privUser struct {
	Username *string `json:"username"`
	Name *string `json:"name"`
	Email *string `json:"email"`
	Admin *bool `json:"admin"`
	PublicKey *string `json:"public_key"`
	Passwd *string `json:"public_key"`
	Salt *[]byte `json:"salt"`
}

// Create a new API user.
func New(name string) (*User, util.Gerror) {
	var found bool
	var err util.Gerror
	if config.UsingDB() {
		var uerr error
		found, uerr = checkForUserSQL(data_store.Dbh, name)
		if uerr != nil {
			err = util.Errorf(uerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := data_store.New()
		_, found = ds.Get("user", name)
	}
	if found {
		err := util.Errorf("User '%s' already exists", name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}

	if err := validateUserName(name); err != nil {
		return nil, err
	}

	salt, saltErr := chef_crypto.GenerateSalt()
	if saltErr != nil {
		err := util.Errorf(saltErr.Error())
		return nil, err
	}
	user := &User{
		Username: name,
		Name: name,
		Admin: false,
		Email: "",
		pubKey: "",
		salt: salt,
	}
	return user, nil
}

// Gets a user.
func Get(name string) (*User, util.Gerror){
	var user *User
	if config.UsingDB() {
		var err error
		if config.Config.UseMySQL {
			user, err = getUserMySQL(name)
		} else if config.Config.UsePostgreSQL {
			user, err = getUserPostgreSQL(name) 
		}
		if err != nil {
			var gerr util.Gerror
			if err != sql.ErrNoRows {
				gerr = util.Errorf(err.Error())
				gerr.SetStatus(http.StatusInternalServerError)
			} else {
				gerr = util.Errorf("Client %s not found", name)
				gerr.SetStatus(http.StatusNotFound)
			}
			return nil, gerr
		}
	} else {
		ds := data_store.New()
		u, found := ds.Get("user", name)
		if !found {
			err := util.Errorf("User %s not found", name)
			return nil, err
		}
		if u != nil {
			user = u.(*User)
		}
	}
	return user, nil
}

// Save the user's current state.
func (u *User) Save() util.Gerror {
	if config.UsingDB() {
		var err util.Gerror
		if config.Config.UseMySQL {
			err = u.saveMySQL()
		} else {
			err = u.savePostgreSQL()
		}
		if err != nil {
			return err
		}
	} else {
		if err := chkInMemClient(u.Username); err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusConflict)
			return gerr
		}
		ds := data_store.New()
		ds.Set("user", u.Username, u)
	}
	return nil
}

// Deletes a user, but will refuse to do so and give an error if it is the last
// administrator user.
func (u *User) Delete() util.Gerror {
	if u.isLastAdmin() {
		err := util.Errorf("Cannot delete the last admin")
		return err
	}
	if config.UsingDB() {
		var err error
		if config.Config.UseMySQL {
			err = u.deleteMySQL()
		} else {
			err = u.deletePostgreSQL()
		}
		if err != nil {
			gerr := util.CastErr(err)
			return gerr
		}
	} else {
		ds := data_store.New()
		ds.Delete("user", u.Username)
	}
	return nil
}

// Renames a user. Save() must be called after this method is used. Will not 
// rename the last administrator user. 
func (u *User) Rename(new_name string) util.Gerror {
	if err := validateUserName(new_name); err != nil {
		return err
	}
	if u.isLastAdmin() {
		err := util.Errorf("Cannot rename the last admin")
		err.SetStatus(http.StatusForbidden)
		return err
	}
	if config.UsingDB() {
		if config.Config.UseMySQL {
			if err := u.renameMySQL(new_name); err != nil {
				return err
			}
		} else if config.Config.UsePostgreSQL {
			if err := u.renamePostgreSQL(new_name); err != nil {
				return err
			}
		}
	} else {
		ds := data_store.New()
		if err := chkInMemClient(new_name); err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusConflict)
			return gerr
		}
		if _, found := ds.Get("user", new_name); found {
			err := util.Errorf("User %s already exists, cannot rename %s", new_name, u.Username)
			err.SetStatus(http.StatusConflict)
			return err
		}
		ds.Delete("client", u.Username)
	}
	u.Username = new_name
	return nil
}

// Build a new user from a JSON object.
func NewFromJson(json_user map[string]interface{}) (*User, util.Gerror) {
	user_name, nerr := util.ValidateAsString(json_user["name"])
	if nerr != nil {
		return nil, nerr
	}
	user, err := New(user_name)
	if err != nil {
		return nil, err
	}
	// check if the password is supplied if this is a user, and fail if
	// it isn't.
	if _, ok := json_user["password"]; !ok {
		err := util.Errorf("Field 'password' missing")
		return nil, err
	}
	err = user.UpdateFromJson(json_user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Update a user from a JSON object, carrying out a bunch of validations inside.
func (u *User)UpdateFromJson(json_user map[string]interface{}) util.Gerror {
	user_name, nerr := util.ValidateAsString(json_user["name"])
	if nerr != nil {
		return nerr
	}
	if u.Username != user_name {
		err := util.Errorf("User name %s and %s from JSON do not match", u.Username, user_name)
		return err
	}

	/* Validations. */
	/* Invalid top level elements */
	valid_elements := []string{ "username", "name", "org_name", "public_key", "private_key", "admin", "password" }
	ValidElem:
	for k, _ := range json_user {
		for _, i := range valid_elements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}
	var verr util.Gerror

	// Check the password first. If it's bad, bail before touching anything
	// else.
	if passwd, ok := json_user["password"]; ok {
		passwd, verr = util.ValidateAsString(passwd)
		if verr != nil {
			return verr
		}
		verr = u.SetPasswd(passwd.(string))
		if verr != nil {
			return verr
		}
	} 

	if admin_val, ok := json_user["admin"]; ok {
		var ab bool
		if ab, verr = util.ValidateAsBool(admin_val); verr != nil {
			// NOTE: may need to tweak this error message depending
			// if this is a user or a client
			verr = util.Errorf("Field 'admin' invalid")
			return verr
		} else if u.Admin && !ab {
			if u.isLastAdmin() {
				verr = util.Errorf("Cannot remove admin status from the last admin")
				verr.SetStatus(http.StatusForbidden)
				return verr
			}
		}
		u.Admin = ab
	}

	return nil
}

// Returns a list of users.
func GetList() []string {
	var user_list []string
	if config.Config.UseMySQL {
		user_list = getListMySQL()
	} else if config.Config.UsePostgreSQL {
		user_list = getListPostgreSQL()
	} else {
		ds := data_store.New()
		user_list = ds.GetList("user")
	}
	return user_list
}

// Convert the user to a JSON object, massaging it as needed to keep the chef
// client happy (be it knife, chef-pedant, etc.) NOTE: There may be a more
// idiomatic way to do this.
func (u *User) ToJson() map[string]interface{} {
	toJson := make(map[string]interface{})
	toJson["name"] = u.Name
	toJson["admin"] = u.Admin
	toJson["public_key"] = u.PublicKey()

	return toJson
}

func (u *User) isLastAdmin() bool {
	if u.Admin {
		numAdmins := 0
		if config.Config.UseMySQL {
			numAdmins = numAdminsMySQL()
		} else if config.Config.UsePostgreSQL {
			numAdmins = numAdminsPostgreSQL()
		} else {		
			user_list := GetList()
			for _, u := range user_list {
				u1, _ := Get(u)
				if u1 != nil && u1.Admin {
					numAdmins++
				}
			}
		}
		if numAdmins == 1 {
			return true
		}
	}
	return false
}

// Generate a new set of RSA keys for the user. The new private key is saved
// with the user object, the public key is given to the user and not saved on 
// the server at all. 
func (u *User) GenerateKeys() (string, error){
	priv_pem, pub_pem, err := chef_crypto.GenerateRSAKeys()
	if err != nil {
		return "", err
	}
	u.pubKey = pub_pem
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

// Is the actor in question the same client or user as the caller?
func (u *User) IsSelf(other interface{}) bool {
	if !config.Config.UseAuth {
		return true
	}
	if ou, ok := other.(*User); ok {
		if u.Username == ou.Username {
			return true
		}
	}
	return false
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

// Set the user's public key. Part of the Actor interface.
func (u *User) SetPublicKey(pk interface{}) error {
	switch pk := pk.(type) {
		case string:
			ok, err := ValidatePublicKey(pk)
			if !ok {
				return err
			}
			u.pubKey = pk
		default:
			err := fmt.Errorf("invalid type %T for public key", pk)
			return err
	}
	return nil
}

func (u *User) CheckPermEdit(user_data map[string]interface{}, perm string) util.Gerror {
	gerr := util.Errorf("You are not allowed to take this action.")
	gerr.SetStatus(http.StatusForbidden)

	if av, ok := user_data[perm]; ok {
		if a, _ := util.ValidateAsBool(av); a {
			return gerr
		}
	}
	return nil
}

// Validate and set the user's password. Will not set a password for a client.
func (u *User) SetPasswd(password string) util.Gerror {
	if len(password) < 6 {
		err := util.Errorf("Password must have at least 6 characters")
		return err
	}
	/* If those validations pass, set the password */
	var perr error
	u.passwd, perr = chef_crypto.HashPasswd(password, u.salt)
	if perr != nil {
		err := util.Errorf(perr.Error())
		return err
	}
	return nil
}

// Check the provided password to see if it matches the stored password hash.
func (u *User) CheckPasswd(password string) util.Gerror {
	h, perr := chef_crypto.HashPasswd(password, u.salt) 
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

func (u *User) export() *privUser {
	return &privUser{ Name: &u.Name, Username: &u.Username, PublicKey: &u.pubKey, Admin: &u.Admin, Email: &u.Email, Passwd: &u.passwd, Salt: &u.salt }
}

func (u *User) GobEncode() ([]byte, error) {
	prv := u.export()
	buf := new(bytes.Buffer)
	decoder := gob.NewEncoder(buf)
	if err := decoder.Encode(prv); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (u *User) GobDecode(b []byte) error {
	prv := u.export()
	buf := bytes.NewReader(b)
	encoder := gob.NewDecoder(buf)
	err := encoder.Decode(prv)
	if err != nil {
		return err
	}

	return nil
}
