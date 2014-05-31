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
Package client defines the Chef clients. Formerly clients and users were the
same kind of object and stored together, but they have now been split apart.
They do both implement the Actor interface, though. Clients are the more usual
case for the nodes interacting with the server, but users are used for the 
webui and often for general user (as opposed to node) interactions with the 
server.
*/
package client

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
	"database/sql"
)

// A client and a user are very similar, with some small differences - users 
// can never be validators, while clients don't have passwords. Generally nodes 
// and the like will be clients, while people interacting with the goiardi 
// server will be users.
type Client struct {
	Name string `json:"name"`
	NodeName string `json:"node_name"`
	JsonClass string `json:"json_class"`
	ChefType string `json:"chef_type"`
	Validator bool `json:"validator"`
	Orgname string `json:"orgname"`
	pubKey string `json:"public_key"`
	Admin bool `json:"admin"`
	Certificate string `json:"certificate"`
}

// for gob encoding. Needed the json tags for flattening, but that's handled
// by a different struct now. However, they're staying because they may still be
// useful.
type privClient struct {
	Name *string `json:"name"`
	NodeName *string `json:"node_name"`
	JsonClass *string `json:"json_class"`
	ChefType *string `json:"chef_type"`
	Validator *bool `json:"validator"`
	Orgname *string `json:"orgname"`
	PublicKey *string `json:"public_key"`
	Admin *bool `json:"admin"`
	Certificate *string `json:"certificate"`
}

// For flattening. Needs the json tags for flattening.
type flatClient struct {
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

// Creates a new client.
func New(clientname string) (*Client, util.Gerror){
	var found bool
	var err util.Gerror
	if config.UsingDB() {
		var cerr error
		found, cerr = checkForClientSQL(data_store.Dbh, clientname)
		if cerr != nil {
			err := util.Errorf(err.Error())
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := data_store.New()
		_, found = ds.Get("client", clientname)
	}
	if found {
		err = util.Errorf("Client already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if err := validateClientName(clientname); err != nil {
		return nil, err
	}
	client := &Client{
		Name: clientname,
		NodeName: clientname,
		ChefType: "client",
		JsonClass: "Chef::ApiClient",
		Validator: false,
		Orgname: "",
		pubKey: "",
		Admin: false,
		Certificate: "",
	}
	return client, nil
}

// Gets an actor from the data store.
func Get(clientname string) (*Client, util.Gerror){
	var client *Client
	var err error

	if config.UsingDB() {
		if config.Config.UseMySQL {
			client, err = getClientMySQL(clientname)
		} else if config.Config.UsePostgreSQL {
			client, err = getClientPostgreSQL(clientname)
		}
		if err != nil {
			var gerr util.Gerror
			if err != sql.ErrNoRows {
				gerr = util.Errorf(err.Error())
				gerr.SetStatus(http.StatusInternalServerError)
			} else {
				gerr = util.Errorf("Client %s not found", clientname)
				gerr.SetStatus(http.StatusNotFound)
			}
			return nil, gerr
		}
	} else {
		ds := data_store.New()
		c, found := ds.Get("client", clientname)
		if !found {
			gerr := util.Errorf("Client %s not found", clientname)
			gerr.SetStatus(http.StatusNotFound)
			return nil, gerr
		}
		if c != nil {
			client = c.(*Client)
		}
	}
	return client, nil
}

// Save the client. If a user with the same name as the client exists, returns
// an error. Additionally, if running with MySQL it will return any DB error.
func (c *Client) Save() error {
	if config.UsingDB() {
		var err error
		if config.Config.UseMySQL {
			err = c.saveMySQL()
		} else if config.Config.UsePostgreSQL {
			err = c.savePostgreSQL()
		}
		if err != nil {
			return err
		}
	} else {
		if err := chkInMemUser(c.Name); err != nil {
			return err
		}
		ds := data_store.New()
		ds.Set("client", c.Name, c)
	}
	indexer.IndexObj(c)
	return nil
}

// Deletes a client, but will refuse to do so if it is the last client
// that is an adminstrator.
func (c *Client) Delete() error {
	// Make sure this isn't the last admin or something
	// This will be a *lot* easier with an actual database.
	if c.isLastAdmin() {
		err := fmt.Errorf("Cannot delete the last admin")
		return err
	}

	if config.UsingDB() {
		var err error
		if config.Config.UseMySQL {
			err = c.deleteMySQL()
		} else if config.Config.UsePostgreSQL {
			err = c.deletePostgreSQL()
		}
		if err != nil {
			return err
		}
	} else {
		ds := data_store.New()
		ds.Delete("client", c.Name)
	}
	indexer.DeleteItemFromCollection("client", c.Name)
	return nil
}

// Convert the client object into a JSON object, massaging it as needed
// to make chef-pedant happy.
func (c *Client) ToJson() map[string]interface{} {
	toJson := make(map[string]interface{})
	toJson["name"] = c.Name
	toJson["admin"] = c.Admin
	toJson["public_key"] = c.PublicKey()
	toJson["validator"] = c.Validator
	toJson["json_class"] = c.JsonClass
	toJson["chef_type"] = c.ChefType

	return toJson
}

func (c *Client) isLastAdmin() bool {
	if c.Admin {
		numAdmins := 0
		if config.Config.UseMySQL {
			numAdmins = numAdminsMySQL()
		} else if config.Config.UsePostgreSQL {
			numAdmins = numAdminsPostgreSQL()
		} else {
			clist := GetList()
			for _, cc := range clist {
				c1, _ := Get(cc)
				if c1 != nil && c1.Admin {
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

// Renames the client. Save() must be called after this method is used.
// Will not rename the last admin.
func (c *Client) Rename(new_name string) util.Gerror {
	if err := validateClientName(new_name); err != nil {
		return err
	}
	if c.isLastAdmin() {
		err := util.Errorf("Cannot rename the last admin")
		err.SetStatus(http.StatusForbidden)
		return err
	}

	if config.UsingDB() {
		var err util.Gerror
		if config.Config.UseMySQL {
			err = c.renameMySQL(new_name)
		} else if config.Config.UsePostgreSQL {
			err = c.renamePostgreSQL(new_name)
		}
		if err != nil {
			return err
		}
	} else {
		if err := chkInMemUser(new_name); err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusConflict)
			return gerr
		}
		ds := data_store.New()
		if _, found := ds.Get("client", new_name); found {
			err := util.Errorf("Client %s already exists, cannot rename %s", new_name, c.Name)
			err.SetStatus(http.StatusConflict)
			return err
		}
		ds.Delete("client", c.Name)
	}
	c.Name = new_name
	return nil
}

// Build a new client/user from a json object.
func NewFromJson(json_actor map[string]interface{}) (*Client, util.Gerror) {
	actor_name, nerr := util.ValidateAsString(json_actor["name"])
	if nerr != nil {
		return nil, nerr
	}
	client, err := New(actor_name)
	if err != nil {
		return nil, err
	}
	err = client.UpdateFromJson(json_actor)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// Update a client/user from a json object. Does a bunch of validations inside
// rather than in the handler.
func (c *Client)UpdateFromJson(json_actor map[string]interface{}) util.Gerror {
	actor_name, nerr := util.ValidateAsString(json_actor["name"])
	if nerr != nil {
		return nerr
	}
	if c.Name != actor_name {
		err := util.Errorf("Client name %s and %s from JSON do not match", c.Name, actor_name)
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
		if json_actor["chef_type"].(string) != "client" {
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
			if c.isLastAdmin() {
				verr = util.Errorf("Cannot remove admin status from the last admin")
				verr.SetStatus(http.StatusForbidden)
				return verr
			}
		}
	}
	if validator_val, ok := json_actor["validator"]; ok {
		if vb, verr = util.ValidateAsBool(validator_val); verr != nil {
			return verr
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

// Returns a list of clients.
func GetList() []string {
	var client_list []string
	if config.Config.UseMySQL {
		client_list = getListMySQL()
	} else if config.Config.UsePostgreSQL {
		client_list = getListPostgreSQL()
	} else {
		ds := data_store.New()
		client_list = ds.GetList("client")
	}
	return client_list
}

// Generate a new set of RSA keys for the client. The new private key is saved
// with the client, the public key is given to the client and not saved on the
// server at all. 
func (c *Client) GenerateKeys() (string, error){
	priv_pem, pub_pem, err := chef_crypto.GenerateRSAKeys()
	if err != nil {
		return "", err
	}
	c.pubKey = pub_pem
	return priv_pem, nil
}

func (a *Client) GetName() string {
	return a.Name
}

func (a *Client) URLType() string {
	url_type := fmt.Sprintf("%ss", a.ChefType)
	return url_type
}

func validateClientName(name string) util.Gerror {
	if !util.ValidateName(name) {
		err := util.Errorf("Invalid client name '%s' using regex: 'Malformed client name.  Must be A-Z, a-z, 0-9, _, -, or .'.", name)
		return err
	}
	return nil
}

/* Search indexing functions */

func (c *Client) DocId() string {
	return c.Name
}

func (c *Client) Index() string {
	return "client"
}

func (c *Client) Flatten() []string {
	flatten := util.FlattenObj(c.flatExport())
	indexified := util.Indexify(flatten)
	return indexified
}

/* Permission functions. Later role-based perms may be implemented, but for now
 * it's just the basic admin/validator/user perms */

// Is the client an admin? If use-auth is false, this always returns true.
func (c *Client) IsAdmin() bool {
	if !useAuth(){
		return true
	}
	return c.Admin
}

// Is the client a validator client? If use-auth is false, this always returns 
// false. 
func (c *Client) IsValidator() bool {
	if !useAuth(){
		return false
	}
	if c.ChefType == "client" && c.Validator {
		return c.Validator
	}
	return false
}

// Is the other actor provided the same as the caller.
func (c *Client) IsSelf(other interface{}) bool {
	if !useAuth(){
		return true
	}
	if oc, ok := other.(*Client); ok {
		if c.Name == oc.Name {
			return true
		}
	}
	return false
}

func (c *Client) IsUser() bool {
	return false
}

func (c *Client) IsClient() bool {
	return true
}

// Returns the client's public key. Part of the Actor interface.
func (c *Client) PublicKey() string {
	return c.pubKey
}

// Set the client's public key.
func (c *Client) SetPublicKey(pk interface{}) error {
	switch pk := pk.(type) {
		case string:
			ok, err := ValidatePublicKey(pk)
			if !ok {
				return err
			}
			c.pubKey = pk
		default:
			err := fmt.Errorf("invalid type %T for public key", pk)
			return err
	}
	return nil
}

// A check to see if the client is trying to edit admin and validator 
// attributes.
func (c *Client) CheckPermEdit(client_data map[string]interface{}, perm string) util.Gerror {
	gerr := util.Errorf("You are not allowed to take this action.")
	gerr.SetStatus(http.StatusForbidden)

	if av, ok := client_data[perm]; ok {
		if a, _ := util.ValidateAsBool(av); a {
			return gerr
		}
	}
	return nil
}

/* a check to see if we should do perm checks */
func useAuth() bool {
	return config.Config.UseAuth
}

func (c *Client) export() *privClient {
	return &privClient{ Name: &c.Name, NodeName: &c.NodeName, JsonClass: &c.JsonClass, ChefType: &c.ChefType, Validator: &c.Validator, Orgname: &c.Orgname, PublicKey: &c.pubKey, Admin: &c.Admin, Certificate: &c.Certificate }
}

func (c *Client) flatExport() *flatClient {
	return &flatClient{ Name: c.Name, NodeName: c.NodeName, JsonClass: c.JsonClass, ChefType: c.ChefType, Validator: c.Validator, Orgname: c.Orgname, PublicKey: c.pubKey, Admin: c.Admin, Certificate: c.Certificate }
}

func (c *Client) GobEncode() ([]byte, error) {
	prv := c.export()
	buf := new(bytes.Buffer)
	decoder := gob.NewEncoder(buf)
	if err := decoder.Encode(prv); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Client) GobDecode(b []byte) error {
	prv := c.export()
	buf := bytes.NewReader(b)
	encoder := gob.NewDecoder(buf)
	err := encoder.Decode(prv)
	if err != nil {
		return err
	}

	return nil
}
