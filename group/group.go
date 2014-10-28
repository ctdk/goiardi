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

package group

import (
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"sync"
)

var DefaultGroups = [4]string{"admins", "billing-admins", "clients", "users"}

type Group struct {
	Name   string
	Org    *organization.Organization
	Actors []actor.Actor
	Groups []*Group
	m      sync.RWMutex
}

func New(org *organization.Organization, name string) (*Group, util.Gerror) {
	// will need to validate group name, presumably

	var found bool
	if config.UsingDB() {

	} else {
		ds := datastore.New()
		_, found = ds.Get(org.DataKey("group"), name)
	}
	if found {
		err := util.Errorf("Group %s in organization %s already exists", name, org.Name)
		return nil, err
	}
	g := &Group{
		Name: name,
		Org:  org,
	}
	return g, nil
}

func Get(org *organization.Organization, name string) (*Group, util.Gerror) {
	if config.UsingDB() {

	}
	ds := datastore.New()
	g, found := ds.Get(org.DataKey("group"), name)
	var group *Group
	if g != nil {
		group = g.(*Group)
	}
	if !found {
		err := util.Errorf("group '%s' not found in organization %s", name, org.Name)
		err.SetStatus(http.StatusNotFound)
		return nil, err
	}
	return group, nil
}

// TODO: functions to safely add/remove actors and groups to/from the group

func (g *Group) Save() util.Gerror {
	if config.UsingDB() {

	}
	ds := datastore.New()
	ds.Set(g.Org.DataKey("group"), g.Name, g)
	return nil
}

func (g *Group) Delete() util.Gerror {
	if config.UsingDB() {

	}
	ds := datastore.New()
	ds.Delete(g.Org.DataKey("group"), g.Name)
	return nil
}

func GetList(org *organization.Organization) []string {
	if config.UsingDB() {

	}
	ds := datastore.New()
	groupList := ds.GetList(org.DataKey("group"))
	return groupList
}

func (g *Group) GetName() string {
	return g.Name
}

func (g *Group) URLType() string {
	return "groups"
}

func (g *Group) OrgName() string {
	return g.Org.Name
}

// should this actually return the groups?

func MakeDefaultGroups(org *organization.Organization) util.Gerror {
	for _, n := range DefaultGroups {
		g, err := New(org, n)
		if err != nil {
			return err
		}
		err = g.Save()
		if err != nil {
			return err
		}
	}
	return nil
}
