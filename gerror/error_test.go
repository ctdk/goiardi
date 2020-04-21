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

package gerror

import (
	"fmt"
	"golang.org/x/xerrors"
	"net/http"
	"testing"
)

var gwrapper = New("christmas wrapping")

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

func TestUnwrap(t *testing.T) {
	if gwrapper.Unwrap() == nil {
		t.Error("Unwrap() method failed")
	}

	zerr := xerrors.Errorf("moar: %w", gwrapper)

	if !xerrors.Is(zerr, gwrapper) {
		t.Error("Somehow zerr was not grwapper")
	}
}

func TestCastErr(t *testing.T) {
	err := fmt.Errorf("an heroic error")
	cerr := CastErr(err)
	var v *gerror
	if !xerrors.As(cerr, &v) {
		t.Error("CastErr did not work properly")
	}

	cerr.SetStatus(http.StatusNotFound)

	berr := CastErr(cerr)
	if berr.Status() != http.StatusNotFound {
		t.Error("CastErr improperly changed the original gerror's status.")
	}
}
