/* The client object */

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

/* 
Package actor in goiardi encompasses both Chef clients and users. They're
basically the same thing. Clients are the more usual case - users are mainly
used for the web interface (which goiardi doesn't have yet). 
*/
package actor

import (
	"github.com/ctdk/goiardi/data_store"
	"fmt"
	"github.com/ctdk/goiardi/chef_crypto"
	"github.com/ctdk/goiardi/util"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/config"
	"net/http"
	"encoding/gob"
	"bytes"
)

type Actor struct {
	Name string `json:"name"`
	NodeName string `json:"node_name"`
	JsonClass string `json:"json_class"`
	ChefType string `json:"chef_type"`
	Validator bool `json:"validator"`
	Orgname string `json:"orgname"`
	PublicKey string `json:"public_key"`
	Admin bool `json:"admin"`
	Certificate string `json:"certificate"`
	passwd string
	Salt []byte
}

// for gob encoding. Needed the json tags for flattening, but that's handled
// by a different struct now. However, they're staying because they may still be
// useful.
type privActor struct {
	Name *string `json:"name"`
	NodeName *string `json:"node_name"`
	JsonClass *string `json:"json_class"`
	ChefType *string `json:"chef_type"`
	Validator *bool `json:"validator"`
	Orgname *string `json:"orgname"`
	PublicKey *string `json:"public_key"`
	Admin *bool `json:"admin"`
	Certificate *string `json:"certificate"`
	Passwd *string `json:"passwd"`
	Salt *[]byte `json:"salt"`
}

// for flattening. Needs the json tags for flattening.
type flatActor struct {
	Name string `json:"name"`
	NodeName string `json:"node_name"`
	JsonClass string `json:"json_class"`
	ChefType string `json:"chef_type"`
	Validator bool `json:"validator"`
	Orgname string `json:"orgname"`
	PublicKey string `json:"public_key"`
	Admin bool `json:"admin"`
	Certificate string `json:"certificate"`
}

func New(clientname string, cheftype string) (*Actor, util.Gerror){
	ds := data_store.New()
	if _, found := ds.Get("client", clientname); found {
		/* Hrmph, needs slightly different messages for clients and
		 * users. */
		var errstr string
		if cheftype == "user" {
			errstr = fmt.Sprintf("User '%s' already exists", clientname)
		} else {
			errstr = "Client already exists"
		}
		err := util.Errorf(errstr)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if err := validateClientName(clientname, cheftype); err != nil {
		return nil, err
	}
	actor := &Actor{
		Name: clientname,
		NodeName: clientname,
		JsonClass: "Chef::ApiClient",
		ChefType: cheftype,
		Validator: false,
		Orgname: "",
		PublicKey: "",
		Admin: false,
		Certificate: "",
	}
	if cheftype == "user" {
		salt, saltErr := chef_crypto.GenerateSalt()
		if saltErr != nil {
			err := util.Errorf(saltErr.Error())
			return nil, err
		}
		actor.Salt = salt
	} else {
		/* May not be strictly necessary, but since this would be set
		 * by the data store when the client is loaded from the data
		 * store anyway, it may as well be set to an empty array. */
		actor.Salt = make([]byte, 0)
	}
	return actor, nil
}


func Get(clientname string) (*Actor, error){
	ds := data_store.New()
	client, found := ds.Get("client", clientname)
	if !found{
		err := fmt.Errorf("Client (or user) %s not found", clientname)
		return nil, err
	}
	return client.(*Actor), nil
}

func GetReqUser(clientname string) (*Actor, util.Gerror) {
	/* If UseAuth is turned off, use the automatically created admin user */
	if !config.Config.UseAuth {
		clientname = "admin"
	}
	c, err := Get(clientname)
	if err != nil {
		/* Theoretically it should be hard to reach this point, since
		 * if the signed request was accepted the user ought to exist.
		 * Still, best to be cautious. */
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusUnauthorized)
		return nil, gerr
	}
	return c, nil
}

func (c *Actor) Save() error {
	ds := data_store.New()
	ds.Set("client", c.Name, c)
	indexer.IndexObj(c)
	return nil
}

func (c *Actor) Delete() error {
	// Make sure this isn't the last admin or something
	// This will be a *lot* easier with an actual database.
	if c.IsLastAdmin() {
		err := fmt.Errorf("Cannot delete the last admin")
		return err
	}
	ds := data_store.New()
	ds.Delete("client", c.Name)
	indexer.DeleteItemFromCollection("client", c.Name)
	return nil
}

// Convert the client or user object into a JSON object, massaging it as needed
// to make chef-pedant happy.
func (c *Actor) ToJson() map[string]interface{} {
	toJson := make(map[string]interface{})
	toJson["name"] = c.Name
	toJson["admin"] = c.Admin
	toJson["public_key"] = c.PublicKey

	if c.ChefType != "user" {
		toJson["validator"] = c.Validator
		toJson["json_class"] = c.JsonClass
		toJson["chef_type"] = c.ChefType
	}

	return toJson
}

func (c *Actor) IsLastAdmin() bool {
	if c.Admin {
		clist := GetList()
		numAdmins := 0
		for _, cc := range clist {
			c1, _ := Get(cc)
			if c1 != nil && c1.Admin && (c1.ChefType == c.ChefType){
				numAdmins++
			}
		}
		if numAdmins == 1 {
			return true
		}
	}
	return false
}

// Renames the client or user. Save() must be called after this method is used.
func (c *Actor) Rename(new_name string) util.Gerror {
	ds := data_store.New()
	if err := validateClientName(new_name, c.ChefType); err != nil {
		return err
	}
	if c.IsLastAdmin() {
		err := util.Errorf("Cannot rename the last admin")
		err.SetStatus(http.StatusForbidden)
		return err
	}
	if _, found := ds.Get("client", new_name); found {
		err := util.Errorf("Client (or user) %s already exists, cannot rename %s", new_name, c.Name)
		err.SetStatus(http.StatusConflict)
		return err
	}
	ds.Delete("client", c.Name)
	c.Name = new_name
	return nil
}

// Build a new client/user from a json object
func NewFromJson(json_actor map[string]interface{}, cheftype string) (*Actor, util.Gerror) {
	actor_name, nerr := util.ValidateAsString(json_actor["name"])
	if nerr != nil {
		return nil, nerr
	}
	actor, err := New(actor_name, cheftype)
	if err != nil {
		return nil, err
	}
	// check if the password is supplied if this is a user, and fail if
	// it isn't.
	if _, ok := json_actor["password"]; !ok && cheftype == "user" {
		err := util.Errorf("Field 'password' missing")
		return nil, err
	}
	err = actor.UpdateFromJson(json_actor, cheftype)
	if err != nil {
		return nil, err
	}
	return actor, nil
}

// Update a client/user from a json object. Does a bunch of validations inside
// rather than in the handler.
func (c *Actor)UpdateFromJson(json_actor map[string]interface{}, cheftype string) util.Gerror {
	actor_name, nerr := util.ValidateAsString(json_actor["name"])
	if nerr != nil {
		return nerr
	}
	if c.Name != actor_name {
		err := util.Errorf("Client (or user) name %s and %s from JSON do not match", c.Name, actor_name)
		return err
	}

	/* Validations. */
	/* Invalid top level elements */
	valid_elements := []string{ "name", "json_class", "chef_type", "validator", "org_name", "public_key", "private_key", "admin", "certificate", "password" }
	ValidElem:
	for k, _ := range json_actor {
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
	if passwd, ok := json_actor["password"]; ok {
		if cheftype != "user" {
			verr = util.Errorf("clients don't have passwords")
			return verr
		}
		passwd, verr = util.ValidateAsString(passwd)
		if verr != nil {
			return verr
		}
		verr = c.SetPasswd(passwd.(string))
		if verr != nil {
			return verr
		}
	} 

	json_actor["json_class"], verr = util.ValidateAsFieldString(json_actor["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			json_actor["json_class"] = c.JsonClass
		} else {
			return verr
		}
	} else {
		if json_actor["json_class"].(string) != "Chef::ApiClient" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}


	json_actor["chef_type"], verr = util.ValidateAsFieldString(json_actor["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			json_actor["chef_type"] = c.ChefType
		} else {
			return verr
		}
	} else {
		if json_actor["chef_type"].(string) != "client" && json_actor["chef_type"].(string) != "user" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	var ab, vb bool
	if admin_val, ok := json_actor["admin"]; ok {
		if ab, verr = util.ValidateAsBool(admin_val); verr != nil {
			// NOTE: may need to tweak this error message depending
			// if this is a user or a client
			verr = util.Errorf("Field 'admin' invalid")
			return verr
		} else if c.Admin && !ab {
			if c.IsLastAdmin() {
				verr = util.Errorf("Cannot remove admin status from the last admin")
				verr.SetStatus(http.StatusForbidden)
				return verr
			}
		}
	}
	if validator_val, ok := json_actor["validator"]; ok {
		if vb, verr = util.ValidateAsBool(validator_val); verr != nil {
			return verr
		} else {
			/* Just set admin flag here */
			if cheftype == "user" {
				verr = util.Errorf("Cannot make a user a validator")
				return verr
			}
		}
	}
	if ab && vb {
		verr = util.Errorf("Client can be either an admin or a validator, but not both.")
		verr.SetStatus(http.StatusBadRequest)
		return verr
	} else {
		c.Admin = ab
		c.Validator = vb
	}
	c.ChefType = json_actor["chef_type"].(string)
	c.JsonClass = json_actor["json_class"].(string)

	return nil
}

func ValidatePublicKey(publicKey interface{}) (bool, util.Gerror) {
	ok, pkerr := chef_crypto.ValidatePublicKey(publicKey)
	var err util.Gerror
	if !ok {
		err = util.Errorf(pkerr.Error())
	}
	return ok, err
}

// Returns a list of actors. Clients and users are stored together, so no user
// can have the same name as an existing client (and vice versa).
func GetList() []string {
	ds := data_store.New()
	client_list := ds.GetList("client")
	return client_list
}

// Generate a new set of RSA keys for the client. The new private key is saved
// with the client, the public key is given to the client and not saved on the
// server at all. */
func (c *Actor) GenerateKeys() (string, error){
	priv_pem, pub_pem, err := chef_crypto.GenerateRSAKeys()
	if err != nil {
		return "", err
	}
	c.PublicKey = pub_pem
	return priv_pem, nil
}

func (a *Actor) GetName() string {
	return a.Name
}

func (a *Actor) URLType() string {
	url_type := fmt.Sprintf("%ss", a.ChefType)
	return url_type
}

func validateClientName(name string, cheftype string) util.Gerror {
	if cheftype == "user" {
		userErr := validateUserName(name)
		return userErr
	}
	if !util.ValidateName(name) {
		err := util.Errorf("Invalid client name '%s' using regex: 'Malformed client name.  Must be A-Z, a-z, 0-9, _, -, or .'.", name)
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

/* Search indexing functions */
func (c *Actor) DocId() string {
	return c.Name
}

func (c *Actor) Index() string {
	return "client"
}

func (c *Actor) Flatten() []string {
	flatten := util.FlattenObj(c.flatExport())
	indexified := util.Indexify(flatten)
	return indexified
}

/* Permission functions. Later role-based perms may be implemented, but for now
 * it's just the basic admin/validator/user perms */

// Is the user an admin? If use-auth is false, this always returns true.
func (c *Actor) IsAdmin() bool {
	if !useAuth(){
		return true
	}
	return c.Admin
}

// Is the user a validator client? If use-auth is false, this always returns 
// false. Users also always return false.
func (c *Actor) IsValidator() bool {
	if !useAuth(){
		return false
	}
	if c.ChefType == "client" && c.Validator {
		return c.Validator
	}
	return false
}

func (c *Actor) IsSelf(other *Actor) bool {
	if !useAuth(){
		return true
	}
	if c.Name == other.Name {
		return true
	}
	return false
}

func (c *Actor) CheckPermEdit(client_data map[string]interface{}, perm string) util.Gerror {
	gerr := util.Errorf("You are not allowed to take this action.")
	gerr.SetStatus(http.StatusForbidden)

	if av, ok := client_data[perm]; ok {
		if a, _ := util.ValidateAsBool(av); a {
			return gerr
		}
	}
	return nil
}

func useAuth() bool {
	return config.Config.UseAuth
}

func (c *Actor) SetPasswd(password string) util.Gerror {
	if c.ChefType != "user" {
		err := util.Errorf("Clients don't have passwords, dawg")
		return err
	}
	if len(password) < 6 {
		err := util.Errorf("Password must have at least 6 characters")
		return err
	}
	/* If those validations pass, set the password */
	var perr error
	c.passwd, perr = chef_crypto.HashPasswd(password, c.Salt)
	if perr != nil {
		err := util.Errorf(perr.Error())
		return err
	}
	return nil
}

func (c *Actor) CheckPasswd(password string) util.Gerror {
	if c.ChefType != "user" {
		err := util.Errorf("Clients still don't have passwords")
		return err
	}
	h, perr := chef_crypto.HashPasswd(password, c.Salt) 
	if perr != nil {
		err := util.Errorf(perr.Error())
		return err
	}
	if c.passwd != h {
		err := util.Errorf("password did not match")
		return err
	}
	
	return nil
}

func (c *Actor) export() *privActor {
	return &privActor{ Name: &c.Name, NodeName: &c.NodeName, JsonClass: &c.JsonClass, ChefType: &c.ChefType, Validator: &c.Validator, Orgname: &c.Orgname, PublicKey: &c.PublicKey, Admin: &c.Admin, Certificate: &c.Certificate, Passwd: &c.passwd, Salt: &c.Salt }
}

func (c *Actor) flatExport() *flatActor {
	return &flatActor{ Name: c.Name, NodeName: c.NodeName, JsonClass: c.JsonClass, ChefType: c.ChefType, Validator: c.Validator, Orgname: c.Orgname, PublicKey: c.PublicKey, Admin: c.Admin, Certificate: c.Certificate }
}

func (c *Actor) GobEncode() ([]byte, error) {
	prv := c.export()
	buf := new(bytes.Buffer)
	decoder := gob.NewEncoder(buf)
	if err := decoder.Encode(prv); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Actor) GobDecode(b []byte) error {
	prv := c.export()
	buf := bytes.NewReader(b)
	encoder := gob.NewDecoder(buf)
	err := encoder.Decode(prv)
	if err != nil {
		return err
	}

	return nil
}
