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

package gerror

import (
	"net/http"
	"testing"
)

func TestGerror(t *testing.T) {
	errmsg := "foo bar"
	err := Errorf(errmsg)
	if err.Error() != errmsg {
		t.Errorf("expected %s to match %s", err.Error(), errmsg)
	}
	if err.Status() != http.StatusBadRequest {
		t.Errorf("err.Status() did not return expected default")
	}
	err.SetStatus(http.StatusNotFound)
	if err.Status() != http.StatusNotFound {
		t.Errorf("SetStatus did not set Status correctly")
	}
}
