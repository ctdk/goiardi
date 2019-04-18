/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jbingham@gmail.com>)
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

package container

import (
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

var DefaultContainers = [9]string{
	"clients",
	"containers",
	"cookbooks",
	"data",
	"environments",
	"groups",
	"nodes",
	"roles",
	"sandboxes",
}

// there has GOT to be more to this
type Container struct {
	Name string
	Org  *organization.Organization
}

func New(org *organization.Organization, name string) (*Container, util.Gerror) {
	var found bool
	if !util.ValidateName(name) {
		return nil, util.Errorf("invalid name '%s' for container", name)
	}
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("container"), name)
	}
	if found {
		err := util.Errorf("Container %s in organization %s already exists", name, org.Name)
		err.SetStatus(http.StatusConflict)
		return nil, err
	}
	c := &Container{
		Name: name,
		Org:  org,
	}
	return c, nil
}

func Get(org *organization.Organization, name string) (*Container, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	c, found := ds.Get(org.DataKey("container"), name)
	var container *Container
	if c != nil {
		container = c.(*Container)
	}
	if !found {
		err := util.Errorf("container '%s' not found in organization %s", name, org.Name)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	container.Org = org
	return container, nil
}

func (c *Container) Save() util.Gerror {
	if config.UsingDB() {

	}
	ds := datastore.New()
	ds.Set(c.Org.DataKey("container"), c.Name, c)
	return nil
}

func (c *Container) Delete() util.Gerror {
	if config.UsingDB() {

	}
	ds := datastore.New()
	ds.Delete(c.Org.DataKey("container"), c.Name)
	if _, err := c.Org.PermCheck.DeleteItemACL(c); err != nil {
		return util.CastErr(err)
	}

	return nil
}

func GetList(org *organization.Organization) []string {
	if config.UsingDB() {

	}
	ds := datastore.New()
	conList := ds.GetList(org.DataKey("container"))
	return conList
}

func (c *Container) GetName() string {
	return c.Name
}

func (c *Container) URLType() string {
	return "containers"
}

func (c *Container) OrgName() string {
	return c.Org.Name
}

func (c *Container) ContainerType() string {
	return c.URLType()
}

func (c *Container) ContainerKind() string {
	return "containers"
}

func MakeDefaultContainers(org *organization.Organization) util.Gerror {
	for _, n := range DefaultContainers {
		c, err := New(org, n)
		if err != nil {
			return err
		}
		err = c.Save()
		if err != nil {
			return err
		}
	}
	return nil
}
