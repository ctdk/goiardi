/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package client

import (
	"time"
)

// WIP

// Key represents a client public key. Chef clients can now have more than one
// set of keys, so these have moved to their own table and type.
type Key struct {
	PublicKey string `json:"public_key"`
	Name string `json:"name"`
	// The expiration_date can be "infinity", which requires a bit of hoop
	// jumping on the golang end.
	ExpirationDate time.Time `json:"expiration_date"`
	client_id int64
	id int64
}
