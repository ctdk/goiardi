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
}

func New(clientname string, cheftype string) (*Actor, util.Gerror){
	ds := data_store.New()
	if _, found := ds.Get("client", clientname); found {
		err := util.Errorf("Client already exists")
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	if err := validateClientName(clientname); err != nil {
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
	ds := data_store.New()
	ds.Delete("client", c.Name)
	indexer.DeleteItemFromCollection("client", c.Name)
	return nil
}

// Renames the client or user. Save() must be called after this method is used.
func (c *Actor) Rename(new_name string) util.Gerror {
	ds := data_store.New()
	if err := validateClientName(new_name); err != nil {
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
			return verr
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

func validateClientName(name string) util.Gerror {
	if !util.ValidateName(name) {
		err := util.Errorf("Invalid client name '%s' using regex: 'Malformed client name.  Must be A-Z, a-z, 0-9, _, -, or .'.", name)
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
	flatten := util.FlattenObj(c)
	indexified := util.Indexify(flatten)
	return indexified
}

/* Permission functions. Later role-based perms may be implemented, but for now
 * it's just the basic admin/validator/user perms */

func (c *Actor) IsAdmin() bool {
	return c.Admin
}

func (c *Actor) IsValidator() bool {
	if c.ChefType == "client" && c.Validator {
		return c.Validator
	}
	return false
}

func (c *Actor) IsSelf(other *Actor) bool {
	if c.Name == other.Name {
		return true
	}
	return false
}
