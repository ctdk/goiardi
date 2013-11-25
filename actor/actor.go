/* The client object */

/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
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
)

type Actor struct {
	Name string
	NodeName string
	JsonClass string
	ChefType string
	Validator bool
	Orgname string
	PublicKey string
	Admin bool
	Certificate string
}

func New(clientname string, cheftype string) (*Actor, error){
	ds := data_store.New()
	if _, found := ds.Get("client", clientname); found {
		err := fmt.Errorf("Client (or user) %s already exists", clientname)
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

func (c *Actor) Save() error {
	ds := data_store.New()
	ds.Set("client", c.Name, c)
	return nil
}

func (c *Actor) Delete() error {
	ds := data_store.New()
	ds.Delete("client", c.Name)
	return nil
}

// Renames the client or user. Save() must be called after this method is used.
func (c *Actor) Rename(new_name string) error {
	ds := data_store.New()
	ds.Delete("client", c.Name)
	c.Name = new_name
	return nil
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
