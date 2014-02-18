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

package authentication

import (
	//"github.com/ctdk/goiardi/chef_crypto"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/util"
	"net/http"
	"io"
	//"io/ioutil"
	"bytes"
)

func CheckHeader(user_id string, r *http.Request) (bool, util.Gerror) {
	user, err := actor.Get(user_id)
	_ = user
/*	if err != nil {
		gerr := util.Errorf(err.Error())
		gerr.SetStatus(http.StatusUnauthorized)
		return false, gerr
	} */
	contentHash := r.Header.Get("X-OPS-CONTENT-HASH")
	if contentHash == "" {
		gerr := util.Errorf("no content hash provided")
		gerr.SetStatus(http.StatusUnauthorized)
		return false, gerr
	}
	var bodyBuf bytes.Buffer 
	_, err = io.Copy(&bodyBuf, r.Body)
	if err != nil {
		gerr := util.Errorf("could not copy body")
		gerr.SetStatus(http.StatusInternalServerError)
		return false, gerr
	}
	




	return true, nil
}
