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

/* Package log_info tracks changes to objects when they're saved, noting the
actor performing the action, what kind of action it was, the time of the change,
the type of object and its id, and a dump of the object's state. */
package log_info

import (
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/config"
	"fmt"
	"time"
)

type LogInfo {
	Actor actor.Actor
	ActorType string
	Time time.Time
	Action string
	ObjectType string
	Object util.GoiardiObj
	ExtendedInfo string
	id int
}
