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

// Package user is the result of users and clients ended up having to be split
// apart after all, once adding the SQL backing started falling into place.
// Users are very similar to clients, except that they are unique across the
// whole server and can log in via the web interface, while clients are only
// unique across an organization and cannot log in over the web. Basically,
// users are generally for something you would do, while a client would be
// associated with a specific node.
//
// Note: At this time, organizations are not implemented, so the difference
// between clients and users is a little less stark.
package user

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/chefcrypto"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

// User is, uh, a user. It's very similar to a Client, but subtly different, as
// explained elsewhere.
type User struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Admin    bool   `json:"admin"`
	pubKey   string
	passwd   string
	salt     []byte
}

type privUser struct {
	Username  *string `json:"username"`
	Name      *string `json:"name"`
	Email     *string `json:"email"`
	Admin     *bool   `json:"admin"`
	PublicKey *string `json:"public_key"`
	Passwd    *string `json:"password"`
	Salt      *[]byte `json:"salt"`
}

// New creates a new API user.
func New(name string) (*User, util.Gerror) {
	var found bool
	var err util.Gerror
	if config.UsingDB() {
		var uerr error
		found, uerr = checkForUserSQL(datastore.Dbh, name)
		if uerr != nil {
			err = util.Errorf(uerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := datastore.New()
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

	salt, saltErr := chefcrypto.GenerateSalt()
	if saltErr != nil {
		err := util.Errorf(saltErr.Error())
		return nil, err
	}
	user := &User{
		Username: name,
		Name:     name,
		Admin:    false,
		Email:    "",
		pubKey:   "",
		salt:     salt,
	}
	return user, nil
}

// Get a user.
func Get(name string) (*User, util.Gerror) {
	var user *User
	if config.UsingDB() {
		var err error
		user, err = getUserSQL(name)
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
		ds := datastore.New()
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
		ds := datastore.New()
		ds.Set("user", u.Username, u)
	}
	return nil
}

// Delete a user, but will refuse to do so and give an error if it is the last
// administrator user.
func (u *User) Delete() util.Gerror {
	if u.isLastAdmin() {
		err := util.Errorf("Cannot delete the last admin")
		return err
	}
	if config.UsingDB() {
		err := u.deleteSQL()
		if err != nil {
			gerr := util.CastErr(err)
			return gerr
		}
	} else {
		ds := datastore.New()
		ds.Delete("user", u.Username)
	}
	return nil
}

// Rename a user. Save() must be called after this method is used. Will not
// rename the last administrator user.
func (u *User) Rename(newName string) util.Gerror {
	if err := validateUserName(newName); err != nil {
		return err
	}
	if u.isLastAdmin() {
		err := util.Errorf("Cannot rename the last admin")
		err.SetStatus(http.StatusForbidden)
		return err
	}
	if config.UsingDB() {
		if config.Config.UseMySQL {
			if err := u.renameMySQL(newName); err != nil {
				return err
			}
		} else if config.Config.UsePostgreSQL {
			if err := u.renamePostgreSQL(newName); err != nil {
				return err
			}
		}
	} else {
		ds := datastore.New()
		if err := chkInMemClient(newName); err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusConflict)
			return gerr
		}
		if _, found := ds.Get("user", newName); found {
			err := util.Errorf("User %s already exists, cannot rename %s", newName, u.Username)
			err.SetStatus(http.StatusConflict)
			return err
		}
		ds.Delete("client", u.Username)
	}
	u.Username = newName
	return nil
}

// NewFromJSON builds a new user from a JSON object.
func NewFromJSON(jsonUser map[string]interface{}) (*User, util.Gerror) {
	userName, nerr := util.ValidateAsString(jsonUser["name"])
	if nerr != nil {
		return nil, nerr
	}
	user, err := New(userName)
	if err != nil {
		return nil, err
	}
	// check if the password is supplied if this is a user, and fail if
	// it isn't.
	if _, ok := jsonUser["password"]; !ok {
		err := util.Errorf("Field 'password' missing")
		return nil, err
	}
	err = user.UpdateFromJSON(jsonUser)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateFromJSON updates a user from a JSON object, carrying out a bunch of
// validations inside.
func (u *User) UpdateFromJSON(jsonUser map[string]interface{}) util.Gerror {
	userName, nerr := util.ValidateAsString(jsonUser["name"])
	if nerr != nil {
		return nerr
	}
	if u.Username != userName {
		err := util.Errorf("User name %s and %s from JSON do not match", u.Username, userName)
		return err
	}

	/* Validations. */
	/* Invalid top level elements */
	validElements := []string{"username", "name", "org_name", "public_key", "private_key", "admin", "password", "email", "salt"}
ValidElem:
	for k := range jsonUser {
		for _, i := range validElements {
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
	if passwd, ok := jsonUser["password"]; ok {
		passwd, verr = util.ValidateAsString(passwd)
		if verr != nil {
			return verr
		}
		if passwd != "" {
			verr = u.SetPasswd(passwd.(string))
			if verr != nil {
				return verr
			}
		}
	}

	if adminVal, ok := jsonUser["admin"]; ok {
		var ab bool
		if ab, verr = util.ValidateAsBool(adminVal); verr != nil {
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

// SetPasswdHash is a utility function to directly set a password hash. Only
// especially useful when importing user data with the -m/--import flags, since
// it's still hashed with the user's salt.
func (u *User) SetPasswdHash(pwhash string) {
	if pwhash != "" {
		u.passwd = pwhash
	}
}

// GetList returns a list of users.
func GetList() []string {
	var userList []string
	if config.UsingDB() {
		userList = getListSQL()
	} else {
		ds := datastore.New()
		userList = ds.GetList("user")
	}
	return userList
}

// ToJSON converts the user to a JSON object, massaging it as needed to keep
// the chef client happy (be it knife, chef-pedant, etc.) NOTE: There may be a
// more idiomatic way to do this.
func (u *User) ToJSON() map[string]interface{} {
	toJSON := make(map[string]interface{})
	toJSON["name"] = u.Name
	toJSON["admin"] = u.Admin
	toJSON["public_key"] = u.PublicKey()

	return toJSON
}

func (u *User) isLastAdmin() bool {
	if u.Admin {
		numAdmins := 0
		if config.UsingDB() {
			numAdmins = numAdminsSQL()
		} else {
			userList := GetList()
			for _, u := range userList {
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

// GenerateKeys generates a new set of RSA keys for the user. The new private
// key is saved with the user object, the public key is given to the user and
// not saved on the server at all.
func (u *User) GenerateKeys() (string, error) {
	privPem, pubPem, err := chefcrypto.GenerateRSAKeys()
	if err != nil {
		return "", err
	}
	u.pubKey = pubPem
	return privPem, nil
}

// ValidatePublicKey checks that the provided public key is valid. Wrapper
// around chefcrypto.ValidatePublicKey(), but with a different error type.
func ValidatePublicKey(publicKey interface{}) (bool, util.Gerror) {
	ok, pkerr := chefcrypto.ValidatePublicKey(publicKey)
	var err util.Gerror
	if !ok {
		err = util.CastErr(pkerr)
	}
	return ok, err
}

// IsAdmin returns true if the user is an admin. If use-auth is false, this
// always returns true.
func (u *User) IsAdmin() bool {
	if !config.Config.UseAuth {
		return true
	}
	return u.Admin
}

// IsValidator always returns false, since users are never validators. This is
// true even if auth mode is not on.
func (u *User) IsValidator() bool {
	return false
}

// IsSelf returns true if the actor in question s the same client or user as the
// caller. Always returns true if use-auth is false.
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

// IsUser returns true for users.
func (u *User) IsUser() bool {
	return true
}

// IsClient returns false for users.
func (u *User) IsClient() bool {
	return false
}

// PublicKey returns the user's public key. Part of the Actor interface.
func (u *User) PublicKey() string {
	return u.pubKey
}

// SetPublicKey does what it says on the tin. Part of the Actor interface.
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

// CheckPermEdit checks to see if the user is trying to edit admin and
// validator attributes, and if it has permissions to do so.
func (u *User) CheckPermEdit(userData map[string]interface{}, perm string) util.Gerror {
	gerr := util.Errorf("You are not allowed to take this action.")
	gerr.SetStatus(http.StatusForbidden)

	if av, ok := userData[perm]; ok {
		if a, _ := util.ValidateAsBool(av); a {
			return gerr
		}
	}
	return nil
}

// SetPasswd validates and sets the user's password. Will not set a password for
// a client.
func (u *User) SetPasswd(password string) util.Gerror {
	if len(password) < 6 {
		err := util.Errorf("Password must have at least 6 characters")
		return err
	}
	/* If those validations pass, set the password */
	var perr error
	u.passwd, perr = chefcrypto.HashPasswd(password, u.salt)
	if perr != nil {
		err := util.Errorf(perr.Error())
		return err
	}
	return nil
}

// CheckPasswd checks the provided password to see if it matches the stored
// password hash.
func (u *User) CheckPasswd(password string) util.Gerror {
	h, perr := chefcrypto.HashPasswd(password, u.salt)
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

// GetName returns the user's name.
func (u *User) GetName() string {
	return u.Username
}

// URLType returns the base element of a user's URL.
func (u *User) URLType() string {
	return "users"
}

func (u *User) export() *privUser {
	return &privUser{Name: &u.Name, Username: &u.Username, PublicKey: &u.pubKey, Admin: &u.Admin, Email: &u.Email, Passwd: &u.passwd, Salt: &u.salt}
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

// AllUsers returns all the users on this server.
func AllUsers() []*User {
	var users []*User
	if config.UsingDB() {
		users = allUsersSQL()
	} else {
		userList := GetList()
		for _, u := range userList {
			us, err := Get(u)
			if err != nil {
				continue
			}
			users = append(users, us)
		}
	}
	return users
}

// ExportAllUsers return all users, in a fashion suitable for exporting.
func ExportAllUsers() []interface{} {
	users := AllUsers()
	export := make([]interface{}, len(users))
	for i, u := range users {
		export[i] = u.export()
	}
	return export
}

func chkInMemClient(name string) error {
	var err error
	ds := datastore.New()
	if _, found := ds.Get("clients", name); found {
		err = fmt.Errorf("a client named %s was found that would conflict with this user", name)
	}
	return err
}
