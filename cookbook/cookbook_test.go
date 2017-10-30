/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

package cookbook

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

const minimalCookPath string = "./minimal-cook.json"

func TestLatestConstrained(t *testing.T) {
	//cbname := "minimal"
	f, err := os.Open(minimalCookPath)
	if err != nil {
		t.Error(err)
	}
	dec := json.NewDecoder(f)
	var mc CookbookVersion
	if derr := dec.Decode(&mc); derr != nil {
		t.Error(derr)
	}
	fmt.Printf("cook: %+v\n", mc)
}
