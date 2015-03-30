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
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goiardi/chefcrypto"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

// A Client and a user are very similar, with some small differences - users
// can never be validators, while clients don't have passwords. Generally nodes
// and the like will be clients, while people interacting with the goiardi
// server will be users.
type Client struct {
	Name        string `json:"name"`
	NodeName    string `json:"node_name"`
	JSONClass   string `json:"json_class"`
	ChefType    string `json:"chef_type"`
	Validator   bool   `json:"validator"`
	Orgname     string `json:"orgname"`
	pubKey      string
	Admin       bool   `json:"admin"`
	Certificate string `json:"certificate"`
}

// for gob encoding. Needed the json tags for flattening, but that's handled
// by a different struct now. However, they're staying because they may still be
// useful.
type privClient struct {
	Name        *string `json:"name"`
	NodeName    *string `json:"node_name"`
	JSONClass   *string `json:"json_class"`
	ChefType    *string `json:"chef_type"`
	Validator   *bool   `json:"validator"`
	Orgname     *string `json:"orgname"`
	PublicKey   *string `json:"public_key"`
	Admin       *bool   `json:"admin"`
	Certificate *string `json:"certificate"`
}

// For flattening. Needs the json tags for flattening.
type flatClient struct {
	Name        string `json:"name"`
	NodeName    string `json:"node_name"`
	JSONClass   string `json:"json_class"`
	ChefType    string `json:"chef_type"`
	Validator   bool   `json:"validator"`
	Orgname     string `json:"orgname"`
	PublicKey   string `json:"public_key"`
	Admin       bool   `json:"admin"`
	Certificate string `json:"certificate"`
}

// New creates a new client.
func New(clientname string) (*Client, util.Gerror) {
	var found bool
	var err util.Gerror
	if config.UsingDB() {
		var cerr error
		found, cerr = checkForClientSQL(datastore.Dbh, clientname)
		if cerr != nil {
			err = util.Errorf(cerr.Error())
			err.SetStatus(http.StatusInternalServerError)
			return nil, err
		}
	} else {
		ds := datastore.New()
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
		Name:        clientname,
		NodeName:    clientname,
		ChefType:    "client",
		JSONClass:   "Chef::ApiClient",
		Validator:   false,
		Orgname:     "",
		pubKey:      "",
		Admin:       false,
		Certificate: "",
	}
	return client, nil
}

// Get gets a client from the data store.
func Get(clientname string) (*Client, util.Gerror) {
	var client *Client
	var err error

	if config.UsingDB() {
		client, err = getClientSQL(clientname)
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
		ds := datastore.New()
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
func (c *Client) Save() util.Gerror {
	if config.UsingDB() {
		var err error
		if config.Config.UseMySQL {
			err = c.saveMySQL()
		} else if config.Config.UsePostgreSQL {
			err = c.savePostgreSQL()
		}
		if err != nil {
			return util.CastErr(err)
		}
	} else {
		if err := chkInMemUser(c.Name); err != nil {
			return err
		}
		ds := datastore.New()
		ds.Set("client", c.Name, c)
	}
	indexer.IndexObj(c)
	return nil
}

// Delete a client, but will refuse to do so if it is the last client
// that is an adminstrator.
func (c *Client) Delete() error {
	// Make sure this isn't the last admin or something
	// This will be a *lot* easier with an actual database.
	if c.isLastAdmin() {
		err := fmt.Errorf("Cannot delete the last admin")
		return err
	}

	if config.UsingDB() {
		err := c.deleteSQL()
		if err != nil {
			return err
		}
	} else {
		ds := datastore.New()
		ds.Delete("client", c.Name)
	}
	indexer.DeleteItemFromCollection("client", c.Name)
	return nil
}

// ToJSON converts the client object into a JSON object, massaging it as needed
// to make chef-pedant happy.
func (c *Client) ToJSON() map[string]interface{} {
	toJSON := make(map[string]interface{})
	toJSON["name"] = c.Name
	toJSON["admin"] = c.Admin
	toJSON["public_key"] = c.PublicKey()
	toJSON["validator"] = c.Validator
	toJSON["json_class"] = c.JSONClass
	toJSON["chef_type"] = c.ChefType

	return toJSON
}

func (c *Client) isLastAdmin() bool {
	if c.Admin {
		numAdmins := 0
		if config.UsingDB() {
			numAdmins = numAdminsSQL()
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

// Rename the client. Save() must be called after this method is used.
// Will not rename the last admin.
func (c *Client) Rename(newName string) util.Gerror {
	if err := validateClientName(newName); err != nil {
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
			err = c.renameMySQL(newName)
		} else if config.Config.UsePostgreSQL {
			err = c.renamePostgreSQL(newName)
		}
		if err != nil {
			return err
		}
	} else {
		if err := chkInMemUser(newName); err != nil {
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusConflict)
			return gerr
		}
		ds := datastore.New()
		if _, found := ds.Get("client", newName); found {
			err := util.Errorf("Client %s already exists, cannot rename %s", newName, c.Name)
			err.SetStatus(http.StatusConflict)
			return err
		}
		ds.Delete("client", c.Name)
	}
	c.Name = newName
	return nil
}

// NewFromJSON builds a new client/user from a json object.
func NewFromJSON(jsonActor map[string]interface{}) (*Client, util.Gerror) {
	actorName, nerr := util.ValidateAsString(jsonActor["name"])
	if nerr != nil {
		return nil, nerr
	}
	client, err := New(actorName)
	if err != nil {
		return nil, err
	}
	err = client.UpdateFromJSON(jsonActor)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// UpdateFromJSON updates a client/user from a json object. Does a bunch of
// validations inside rather than in the handler.
func (c *Client) UpdateFromJSON(jsonActor map[string]interface{}) util.Gerror {
	actorName, nerr := util.ValidateAsString(jsonActor["name"])
	if nerr != nil {
		return nerr
	}
	if c.Name != actorName {
		err := util.Errorf("Client name %s and %s from JSON do not match", c.Name, actorName)
		return err
	}

	/* Validations. */
	/* Invalid top level elements */
	validElements := []string{"name", "json_class", "chef_type", "validator", "org_name", "orgname", "public_key", "private_key", "admin", "certificate", "password", "node_name", "clientname"}
ValidElem:
	for k := range jsonActor {
		for _, i := range validElements {
			if k == i {
				continue ValidElem
			}
		}
		err := util.Errorf("Invalid key %s in request body", k)
		return err
	}
	var verr util.Gerror

	jsonActor["json_class"], verr = util.ValidateAsFieldString(jsonActor["json_class"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonActor["json_class"] = c.JSONClass
		} else {
			return verr
		}
	} else {
		if jsonActor["json_class"].(string) != "Chef::ApiClient" {
			verr = util.Errorf("Field 'json_class' invalid")
			return verr
		}
	}

	jsonActor["chef_type"], verr = util.ValidateAsFieldString(jsonActor["chef_type"])
	if verr != nil {
		if verr.Error() == "Field 'name' nil" {
			jsonActor["chef_type"] = c.ChefType
		} else {
			return verr
		}
	} else {
		if jsonActor["chef_type"].(string) != "client" {
			verr = util.Errorf("Field 'chef_type' invalid")
			return verr
		}
	}

	var ab, vb bool
	if adminVal, ok := jsonActor["admin"]; ok {
		if ab, verr = util.ValidateAsBool(adminVal); verr != nil {
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
	if validatorVal, ok := jsonActor["validator"]; ok {
		if vb, verr = util.ValidateAsBool(validatorVal); verr != nil {
			return verr
		}
	}
	if ab && vb {
		verr = util.Errorf("Client can be either an admin or a validator, but not both.")
		verr.SetStatus(http.StatusBadRequest)
		return verr
	}
	c.Admin = ab
	c.Validator = vb

	c.ChefType = jsonActor["chef_type"].(string)
	c.JSONClass = jsonActor["json_class"].(string)

	return nil
}

// ValidatePublicKey checks that the provided public key is valid. Wrapper around
// chefcrypto.ValidatePublicKey(), but with a different error type.
func ValidatePublicKey(publicKey interface{}) (bool, util.Gerror) {
	ok, pkerr := chefcrypto.ValidatePublicKey(publicKey)
	var err util.Gerror
	if !ok {
		err = util.CastErr(pkerr)
	}
	return ok, err
}

// GetList returns a list of clients.
func GetList() []string {
	var clientList []string
	if config.UsingDB() {
		clientList = getListSQL()
	} else {
		ds := datastore.New()
		clientList = ds.GetList("client")
	}
	return clientList
}

// GenerateKeys makes a new set of RSA keys for the client. The new private key
// is saved with the client, the public key is given to the client and not saved
// on the server at all.
func (c *Client) GenerateKeys() (string, error) {
	privPem, pubPem, err := chefcrypto.GenerateRSAKeys()
	if err != nil {
		return "", err
	}
	c.pubKey = pubPem
	return privPem, nil
}

// GetName gives the client's name.
func (c *Client) GetName() string {
	return c.Name
}

// URLType returns the base URL element for clients.
func (c *Client) URLType() string {
	urlType := fmt.Sprintf("%ss", c.ChefType)
	return urlType
}

func validateClientName(name string) util.Gerror {
	if !util.ValidateName(name) {
		err := util.Errorf("Invalid client name '%s' using regex: 'Malformed client name.  Must be A-Z, a-z, 0-9, _, -, or .'.", name)
		return err
	}
	return nil
}

/* Search indexing functions */

// DocID is for the indexer, to tell what it's identifier is.
func (c *Client) DocID() string {
	return c.Name
}

// Index tells the indexer which collection of objects to place the client into.
func (c *Client) Index() string {
	return "client"
}

// Flatten out the client so it's suitable for indexing.
func (c *Client) Flatten() []string {
	flatten := util.FlattenObj(c.flatExport())
	indexified := util.Indexify(flatten)
	return indexified
}

/* Permission functions. Later role-based perms may be implemented, but for now
 * it's just the basic admin/validator/user perms */

// IsAdmin returns true if the client is an admin. If use-auth is false, this
// always returns true.
func (c *Client) IsAdmin() bool {
	if !useAuth() {
		return true
	}
	return c.Admin
}

// IsValidator returns true if the client is a validator client. If use-auth is
// false, this always returns false.
func (c *Client) IsValidator() bool {
	if !useAuth() {
		return false
	}
	if c.ChefType == "client" && c.Validator {
		return c.Validator
	}
	return false
}

// IsSelf returns true if the other actor provided is the same as the caller.
func (c *Client) IsSelf(other interface{}) bool {
	if !useAuth() {
		return true
	}
	if oc, ok := other.(*Client); ok {
		if c.Name == oc.Name {
			return true
		}
	}
	return false
}

// IsUser always returns false for clients. Part of the Actor interface.
func (c *Client) IsUser() bool {
	return false
}

// IsClient always returns true for clients. Part of the Actor interface.
func (c *Client) IsClient() bool {
	return true
}

// PublicKey returns the client's public key. Part of the Actor interface.
func (c *Client) PublicKey() string {
	return c.pubKey
}

// SetPublicKey sets the client's public key.
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

// CheckPermEdit checks to see if the client is trying to edit admin and
// validator attributes, and if it has permissions to do so.
func (c *Client) CheckPermEdit(clientData map[string]interface{}, perm string) util.Gerror {
	gerr := util.Errorf("You are not allowed to take this action.")
	gerr.SetStatus(http.StatusForbidden)

	if av, ok := clientData[perm]; ok {
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
	return &privClient{Name: &c.Name, NodeName: &c.NodeName, JSONClass: &c.JSONClass, ChefType: &c.ChefType, Validator: &c.Validator, Orgname: &c.Orgname, PublicKey: &c.pubKey, Admin: &c.Admin, Certificate: &c.Certificate}
}

func (c *Client) flatExport() *flatClient {
	return &flatClient{Name: c.Name, NodeName: c.NodeName, JSONClass: c.JSONClass, ChefType: c.ChefType, Validator: c.Validator, Orgname: c.Orgname, PublicKey: c.pubKey, Admin: c.Admin, Certificate: c.Certificate}
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

// AllClients returns a slice of all the clients on this server.
func AllClients() []*Client {
	var clients []*Client
	if config.UsingDB() {
		clients = allClientsSQL()
	} else {
		clientList := GetList()
		for _, c := range clientList {
			cl, err := Get(c)
			if err != nil {
				continue
			}
			clients = append(clients, cl)
		}
	}
	return clients
}

// ExportAllClients returns all clients in a fashion suitable for exporting.
func ExportAllClients() []interface{} {
	clients := AllClients()
	export := make([]interface{}, len(clients))
	for i, c := range clients {
		export[i] = c.export()
	}
	return export
}

func chkInMemUser(name string) util.Gerror {
	var err util.Gerror
	ds := datastore.New()
	if _, found := ds.Get("user", name); found {
		err = util.Errorf("a user named %s was found that would conflict with this client", name)
		err.SetStatus(http.StatusConflict)
	}
	return err
}
