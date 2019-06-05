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

// SQL goodies for groups

// Arrrgh, that's right. I need to look up the selecting an array aggregate with
// table join and so forth for groups

/**************

Phew, figured out the query to use to get groups & their members. Here it is
for reference:

-----
select name, organization_id, u.user_ids, c.client_ids, mg.group_ids FROM groups LEFT JOIN 
	(select gau.group_id AS ugid, array_agg(gau.user_id) AS user_ids FROM group_actor_users gau join groups g ON g.id = gau.group_id group by gau.group_id) u ON u.ugid = groups.id 
 LEFT JOIN 
	(select gac.group_id AS cgid, array_agg(gac.client_id) AS client_ids FROM group_actor_clients gac join groups g ON g.id = gac.group_id group by gac.group_id) c ON c.cgid = groups.id
 LEFT JOIN 
	(select gg.group_id AS ggid, array_agg(gg.member_group_id) AS group_ids FROM group_groups gg join groups g ON g.id = gg.group_id group by gg.group_id) mg ON mg.ggid = groups.id
WHERE groups.id = 1;
-----

It does, of course, need some cleaning up.

***************/
