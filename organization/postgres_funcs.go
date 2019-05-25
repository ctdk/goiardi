/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package organization

import (
	"github.com/ctdk/goiardi/util"
)

/*
 * Postgres specific functions for organizations. It's still up in the air if
 * MySQL will come along for the ride to 1.0.0, but we'll see.
 */

func (o *Organization) savePostgreSQL() util.Gerror {
	return nil
}

func (o *Organization) renamePostgreSQL(newName string) util.Gerror {
	return nil
}

func (o *Organization) createSearchSchema() error {
	return nil
}
