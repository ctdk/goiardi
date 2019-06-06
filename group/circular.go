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

package group

import (

)

/*
 * We desperately want to avoid circular reference loops with group membership,
 * where one group has another group as a member, and that group in turn has
 * the first group as a member. This also applies if there's more levels than
 * just the one (e.g a -> b -> c -> a). 
 *
 * Thus, we need some way to check for those circular references. The same
 * depgraph stuff cookbooks use for dependency checking (although with the exact
 * same methods) might be the way to go, but since checking the group membership
 * is much simpler than it is with the cookbook dependencies and version
 * constraints (and because I'm not sure that cookbooks can't have circular
 * dependencies, now that I think about it, even though it's Not a Good
 * Pattern™), the first thing to try is a simple map to store whether we've seen
 * a group already.
 */

func (g *Group) circularCheck() (bool, error) {

}
