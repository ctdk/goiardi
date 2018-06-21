/*
 * Copyright (c) 2013-2018, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package acl

import (
	"strings"
)

// Define the casbin RBAC model and the skeletal default policy.

const modelDefinition = strings.TrimSpace(`
[request_definition]
r = sub, obj, kind, subkind, act

[policy_definition]
p = sub, obj, kind, subkind, act, eft

[role_definition]
g = _, _, _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub, r.kind, r.subkind) && r.kind == p.kind && r.subkind == p.subkind && r.obj == p.obj && r.act == p.act || r.sub == "pivotal"
`)

// NOTE: MySQL/Postgres implementations of this may require some mild heroics
// to put convert this to a form suitable to put in the DB. We'll see what ends
// up happening.

const policySkelDefinition = strings.TrimSpace(`
p, admins, containers, containers, default, create, allow
p, admins, containers, containers, default, read, allow
p, users, containers, containers, default, read, allow
p, admins, containers, containers, default, update, allow
p, admins, containers, containers, default, delete, allow
p, admins, containers, containers, default, grant, allow
p, users, containers, containers, clients, delete, allow
p, users, containers, containers, nodes, create, allow
p, users, containers, containers, environments, create, allow

p, admins, groups, containers, default, create, allow
p, admins, groups, containers, default, read, allow
p, admins, groups, containers, default, update, allow
p, admins, groups, containers, default, delete, allow
p, admins, groups, containers, default, grant, allow
p, users, groups, containers, clients, read, deny

p, admins, cookbooks, containers, default, create, allow
p, admins, cookbooks, containers, default, read, allow
p, admins, cookbooks, containers, default, update, allow
p, admins, cookbooks, containers, default, delete, allow
p, admins, cookbooks, containers, default, grant, allow
p, users, cookbooks, containers, default, create, allow
p, users, cookbooks, containers, default, read, allow
p, users, cookbooks, containers, default, update, allow
p, users, cookbooks, containers, default, delete, allow
p, clients, cookbooks, containers, default, read, allow

p, admins, environments, containers, default, create, allow
p, admins, environments, containers, default, read, allow
p, admins, environments, containers, default, update, allow
p, admins, environments, containers, default, delete, allow
p, admins, environments, containers, default, grant, allow
p, users, environments, containers, default, create, allow
p, users, environments, containers, default, read, allow
p, users, environments, containers, default, update, allow
p, users, environments, containers, default, delete, allow
p, clients, environments, containers, default, read, allow

p, admins, roles, containers, default, create, allow
p, admins, roles, containers, default, read, allow
p, admins, roles, containers, default, update, allow
p, admins, roles, containers, default, delete, allow
p, admins, roles, containers, default, grant, allow
p, users, roles, containers, default, create, allow
p, users, roles, containers, default, read, allow
p, users, roles, containers, default, update, allow
p, users, roles, containers, default, delete, allow
p, clients, roles, containers, default, read, allow

p, admins, data, containers, default, create, allow
p, admins, data, containers, default, read, allow
p, admins, data, containers, default, update, allow
p, admins, data, containers, default, delete, allow
p, admins, data, containers, default, grant, allow
p, users, data, containers, default, create, allow
p, users, data, containers, default, read, allow
p, users, data, containers, default, update, allow
p, users, data, containers, default, delete, allow
p, clients, data, containers, default, read, allow

p, admins, nodes, containers, default, create, allow
p, admins, nodes, containers, default, read, allow
p, admins, nodes, containers, default, update, allow
p, admins, nodes, containers, default, delete, allow
p, admins, nodes, containers, default, grant, allow
p, users, nodes, containers, default, create, allow
p, users, nodes, containers, default, read, allow
p, users, nodes, containers, default, update, allow
p, users, nodes, containers, default, delete, allow
p, clients, nodes, containers, default, create, allow
p, clients, nodes, containers, default, read, allow

p, admins, clients, containers, default, create, allow
p, admins, clients, containers, default, read, allow
p, admins, clients, containers, default, update, allow
p, admins, clients, containers, default, delete, allow
p, admins, clients, containers, default, grant, allow
p, users, clients, containers, default, read, allow
p, users, clients, containers, default, delete, allow

p, admins, sandboxes, containers, default, create, allow
p, admins, sandboxes, containers, default, read, allow
p, admins, sandboxes, containers, default, update, allow
p, admins, sandboxes, containers, default, delete, allow
p, admins, sandboxes, containers, default, grant, allow
p, users, sandboxes, containers, default, create, allow

p, admins, log-infos, containers, default, create, allow
p, admins, log-infos, containers, default, read, allow
p, admins, log-infos, containers, default, update, allow
p, admins, log-infos, containers, default, delete, allow
p, admins, log-infos, containers, default, grant, allow
p, users, log-infos, containers, default, create, allow

p, admins, reports, containers, default, create, allow
p, admins, reports, containers, default, read, allow
p, admins, reports, containers, default, update, allow
p, admins, reports, containers, default, delete, allow
p, admins, reports, containers, default, grant, allow
p, users, reports, containers, default, create, allow
p, clients, reports, containers, default, create, allow

p, admins, shoveys, containers, default, create, allow
p, admins, shoveys, containers, default, read, allow
p, admins, shoveys, containers, default, update, allow
p, admins, shoveys, containers, default, delete, allow
p, admins, shoveys, containers, default, grant, allow
p, clients, shoveys, containers, default, update, allow

p, billing-admins, billing-admins, groups, default, read, allow
p, billing-admins, billing-admins, groups, default, update, allow

p, admins, admins, groups, default, create, allow
p, admins, admins, groups, default, read, allow
p, admins, admins, groups, default, update, allow
p, admins, admins, groups, default, delete, allow
p, admins, admins, groups, default, grant, allow

p, admins, clients, groups, default, create, allow
p, admins, clients, groups, default, read, allow
p, admins, clients, groups, default, update, allow
p, admins, clients, groups, default, delete, allow
p, admins, clients, groups, default, grant, allow

p, admins, users, groups, default, create, allow
p, admins, users, groups, default, read, allow
p, admins, users, groups, default, update, allow
p, admins, users, groups, default, delete, allow
p, admins, users, groups, default, grant, allow

p, admins, default, groups, default, create, allow
p, admins, default, groups, default, read, allow
p, admins, default, groups, default, update, allow
p, admins, default, groups, default, delete, allow
p, admins, default, groups, default, grant, allow
p, users, default, groups, default, read, allow
`)
