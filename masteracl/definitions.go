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

package masteracl

import ()

// The model and default policy for these master perms are, at least, a lot
// simpler than the normal model and policy definitions are.

const modelDefinition = `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

// The policy for permissions affecting goiardi overall need to be stored in a
// separate file (or, someday, in the postgres db).
const masterPolicySkel = `p, $$master_admins, organizations, create
p, $$master_admins, organizations, read
p, $$master_admins, organizations, update
p, $$master_admins, organizations, delete
p, $$master_admins, organizations, grant

p, $$master_admins, reindex, create
p, $$master_admins, reindex, read
p, $$master_admins, reindex, update
p, $$master_admins, reindex, delete
p, $$master_admins, reindex, grant

p, $$master_admins, users, create
p, $$master_admins, users, read
p, $$master_admins, users, update
p, $$master_admins, users, delete
p, $$master_admins, users, grant

g, pivotal, $$master_admins
`
