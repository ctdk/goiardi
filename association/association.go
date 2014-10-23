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

package association

import (
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"net/http"
)

type Association struct {
	User *user.User
	Org *organization.Organization
}

func (a *Association) Key() {
	return util.JoinStr(a.User.Name, "-", a.org.Name)
}

func Set(user *user.User, org *organization.Organization) (*association.Association, util.Gerror) {

}

func Get(key string) (*association.Association, util.Gerror) {

}

func Orgs(user *user.User) ([]*organization.Organization, util.Gerror) {

}

func Users(org *organization.Organization) ([]*user.Users, util.Gerror) {

}
