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

package main

import (
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/gerror"
	"github.com/ctdk/goiardi/util"
	"github.com/tideland/golib/logger"
	"net/http"
)

// Functions and types for HEAD responses for various endpoints

// Implements Chef RFCnnn (unassigned so far)

// Full table of what everything means will be added when the RFC is finalized

type headChecker interface {
	actor.Actor
	DoesExist(string) (bool, util.Gerror)
}

type exists func(resource string) (bool, util.Gerror)

type permChecker func(r *http.Request, resource string, obj actor.Actor) util.Gerror

func headResponse(w http.ResponseWriter, r *http.Request, status int) {
	logger.Infof("HEAD response for status %d", status)
	w.WriteHeader(status)
	return
}

func headChecking(w http.ResponseWriter, r *http.Request, obj actor.Actor, resource string, doesExist exists, permCheck permChecker) {
	found, err := doesExist(resource)
	if err != nil {
		headResponse(w, r, err.Status())
	}
	err = permCheck(r, resource, obj)
	if err != nil {
		headResponse(w, r, err.Status())
	}
	if !found {
		headResponse(w, r, http.StatusNotFound)
	}
	headResponse(w, r, http.StatusOK)
}

func headForbidden() util.Gerror {
	err := gerror.StatusError("forbidden", http.StatusForbidden)
	return err
}
