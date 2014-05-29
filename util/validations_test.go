/*
 * Copyright (c) 2013-2014, Yasushi Abe
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

package util

import (
	"testing"
)

func TestValidateRunList(t *testing.T) {
	runList := []string{
		"recipe[qualified_recipe_name]",
		"recipe[versioned_qualified_recipe_name@1.0.0]",
		"recipe[versioned_qualified_recipe_name2@1.0]",
		"recipe[qualified_recipe_name::include.period]",
		"recipe[versioned_qualified_recipe_name::include.period@1.0.0]",
		"role[qualified_role_name]",
		"versioned_unqualified_recipe_name@1.0.0",
		"unqualified_recipe_name",
	}
	rl, err := ValidateRunList(runList)
	if err != nil {
		t.Errorf("%v shoud have passed run list validation, but didn't.", runList)
	}
	t.Logf("%+v", rl)

	falseFriends := []string{
		"Recipe[recipe_name]",
		"roles[role_name]",
		"recipe[invalid_version@1]",
		"recipe[invalid_version@1.2.3.4]",
		"recipe[invalid_version@abc]",
	}
	for _, falseFriend := range falseFriends {
		if _, err := ValidateRunList([]string{falseFriend}); err == nil {
			t.Errorf("%v shoud not have passed run list validation, but somehow did", falseFriend)
		}
	}
}
