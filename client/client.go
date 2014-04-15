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

// Creates a new actor.
func New(clientname string) (*Client, util.Gerror){
	if config.Config.UseMySQL {
		_, err := data_store.CheckForOne(data_store.Dbh, "clients", clientname)
		if err == nil {
			gerr := util.Errorf("Client already exits")
			gerr.SetStatus(http.StatusConflict)
		} else {
			if err != sql.ErrNoRows {
				gerr := util.Errorf(err.Error())
				gerr.SetStatus(http.StatusInternalServerError)
				return nil, gerr
			}
		}
	} else {
		ds := data_store.New()
		if _, found := ds.Get("client", clientname); found {
			err := util.Errorf("Client already exists")
			err.SetStatus(http.StatusConflict)
			return nil, err
		}
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

func (c *Client) fillClientFromSQL(row *sql.Row) error {
	if config.Config.UseMySQL {
		err := row.Scan(&c.Name, &c.NodeName, &c.Validator, &c.Admin, &c.Orgname, &c.pubKey, &c.Certificate)
		if err != nil {
			return err
		}
		c.ChefType = "client"
		c.JsonClass = "Chef::ApiClient"
	} else {
		err := fmt.Errorf("no database configured, operating in in-memory mode -- fillClientFromSQL cannot be run")
		return err
	}
	return nil
}

// Gets an actor from the data store.
func Get(clientname string) (*Client, error){
	var client *Client
	var found bool

	if config.Config.UseMySQL {
		client = new(Client)
		stmt, err := data_store.Dbh.Prepare("select c.name, nodename, validator, admin, o.name, public_key, certificate FROM clients c JOIN organizations o on c.org_id = o.id WHERE c.name = ?")
		if err != nil {
			return nil, err
		}
		defer stmt.Close()
		row := stmt.QueryRow(clientname)
		err = client.fillClientFromSQL(row)
		if err != nil {
			if err == sql.ErrNoRows {
				found = false
			} else {
				return nil, err
			}
		} else {
			found = true
		}
	} else {
		var c interface{}
		ds := data_store.New()
		c, found = ds.Get("client", clientname)
		client = c.(*Client)
	}
	if !found{
		err := fmt.Errorf("Client %s not found", clientname)
		return nil, err
	}
	return client.(*Client), nil
}

func (c *Client) Save() error {
	if config.Config.UseMySQL {
		tx, err := data_store.Dbh.Begin()
		var client_id int32
		if err != nil {
			return err
		}
		// check for a user with this name first. If orgs are ever
		// implemented, it will only need to check for a user 
		// associated with this organization
		err = chkForUser(tx, c.Name)
		if err != nil
			return err
		}
		client_id, err = data_store.CheckForOne(tx, "clients", c.Name)
		if err == nil {
			_, err := tx.Exec("UPDATE clients SET name = ?, nodename = ?, validator = ?, admin = ?, public_key = ?, certificate = ?, updated_at = NOW() WHERE id = ?", c.Name, c.NodeName, c.Validator, c.Admin, c.pubKey, c.Certificate, client_id)
			if err != nil {
				tx.Rollback()
				return err
			}
		} else {
			if err != sql.ErrNoRows {
				tx.Rollback()
				return err
			}
			_, err = tx.Exec("INSERT INTO clients (name, nodename, validator, admin, public_key, certificate, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())", c.Name, c.NodeName, c.Validator, c.Admin, c.pubKey(), c.Certificate)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
		tx.Commit()
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

// Deletes a client or user, but will refuse to do so if it is the last
// adminstrator of that type.
func (c *Client) Delete() error {
	// Make sure this isn't the last admin or something
	// This will be a *lot* easier with an actual database.
	if c.isLastAdmin() {
		err := fmt.Errorf("Cannot delete the last admin")
		return err
	}

	if config.Config.UseMySQL {
		tx, err := data_store.Dbh.Begin()
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM clients WHERE name = ?", c.Name)
		if err != nil {
			tx.Rollback()
			return err
		}
		tx.Commit()
	} else {
		ds := data_store.New()
		ds.Delete("client", c.Name)
	}
	indexer.DeleteItemFromCollection("client", c.Name)
	return nil
}

// Convert the client or user object into a JSON object, massaging it as needed
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
			stmt, err := data_store.Dbh.Prepare("SELECT count(*) FROM clients WHERE admin = 1")
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()
			err = stmt.QueryRow().Scan(&numAdmins)
			if err != nil {
				log.Fatal(err)
			}
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

// Renames the client or user. Save() must be called after this method is used.
// Will not rename the last admin.
func (c *Client) Rename(new_name string) util.Gerror {
	if err := validateClientName(new_name); err != nil {
		return gerr
	}
	if c.isLastAdmin() {
		err := util.Errorf("Cannot rename the last admin")
		err.SetStatus(http.StatusForbidden)
		return err
	}

	if config.Config.UseMySQL {
		tx, err = data_store.Dbh.Begin()
		if err != nil {
			gerr := util.Errorf(err.Error())
			return gerr
		}
		if err = chkForUser(tx, new_name); err != nil {
			tx.Rollback()
			gerr := util.Errorf(err.Error())
			return gerr
		}
		_, err := data_store.CheckForOne(data_store.Dbh, "clients", clientname)
		if err != sql.ErrNoRows {
			tx.Rollback()
			if err == nil {
				gerr := util.Errorf("Client %s already exists, cannot rename %s", new_name, c.Name)
				gerr.SetStatus(http.StatusConflict)
				return gerr
			} else {
				gerr := util.Errorf(err.Error())
				gerr.SetStatus(http.StatusInternalServerError)
				return gerr
			}
		}
		_, err := tx.Exec("UPDATE clients SET name = ? WHERE name = ?", new_name, c.Name)
		if err != nil {
			tx.Rollback()
			gerr := util.Errorf(err.Error())
			gerr.SetStatus(http.StatusInternalServerError)
			return gerr
		}
		tx.Commit()
	} else {
		if err := chkInMemUser(new_name); err != nil {
			return err
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

// Build a new client/user from a json object
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

// Returns a list of actors. Clients and users are stored together, so no user
// can have the same name as an existing client (and vice versa).
func GetList() []string {
	var client_list []string
	if config.Config.UseMySQL {
		rows, err := data_store.Dbh.QueryRow("SELECT name FROM clients")
		if err != nil {
			if err != sql.ErrNoRows {
				log.Fatal(err)
			}
			rows.Close()
			return client_list
		}
		client_list = make([]string, 0)
		for rows.Next() {
			var client_name string
			err = rows.Scan(&client_name)
			if err != nil {
				log.Fatal(err)
			}
			client_list = append(client_list, client_name)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			log.Fatal(err)
		}
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

// Is the user an admin? If use-auth is false, this always returns true.
func (c *Client) IsAdmin() bool {
	if !useAuth(){
		return true
	}
	return c.Admin
}

// Is the user a validator client? If use-auth is false, this always returns 
// false. Users also always return false.
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

func (c *Client) PublicKey() string {
	return c.pubKey
}

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

// A check to see if the actor is trying to edit admin and validator attributes.
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

func chkForUser(handle data_store.Dbhandle, name string) error {
	var user_id int32
	err = handle.QueryRow("SELECT id FROM users WHERE name = ?", name).Scan(&user_id)
	if err != sql.ErrNoRows {
		if err == nil {
			err = fmt.Errorf("a user with id %d named %s was found that would conflict with this client", user_id, name)
		}
	} else {
		err = nil
	}
	return err 

func chkInMemUser (name string) error {
	var error err
	ds := data_store.New()
	if _, found := ds.Get("users", name); found {
		err = fmt.Errorf("a user named %s was found that would conflict with this client", name)
	}
	return err
}
