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

// Users and clients ended up having to be split apart after all, once adding
// the SQL backing started falling into place. Users are very similar to
// clients, except that they are unique across the whole server and can log in 
// via the web interface, while clients are only unique across an organization
// and cannot log in over the web. Basically, users are generally for something
// you would do, while a client would be associated with a specific node.
//
// Note: At this time, organizations are not implemented, so the difference
// between clients and users is a little less stark.
package user

type User struct {
	Username string `json:"username"`
	Name string `json:"name"`
	Email string `json:"email"`
	Admin bool `json:"admin"`
	PublicKey string `json:"public_key"`
	passwd string
	Salt []byte
}
