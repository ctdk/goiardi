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

// Package gerror defines a custom error type with a HTTP status code for
// goiardi. This used to be in the util package, but has been moved out into its
// own module. For convenience, and because the old methods are called
// everywhere, there are wrappers in util for the Error interface here, along
// with the Errorf and CastErr functions.
package gerror

import (
	"fmt"
	"net/http"
)

// the private error struct
type gerror struct {
	msg    string
	status int
}

// Error is an error type that includes an http status code (defaults to
// http.BadRequest).
type Error interface {
	String() string
	Error() string
	Status() int
	SetStatus(int)
}

// New makes a new Error. Usually you want Errorf.
func New(text string) Error {
	return &gerror{msg: text,
		status: http.StatusBadRequest,
	}
}

// Errorf creates a new Error, with a formatted error string.
func Errorf(format string, a ...interface{}) Error {
	return New(fmt.Sprintf(format, a...))
}

// CastErr will easily cast a different kind of error to a goiardi Error.
func CastErr(err error) Error {
	return Errorf(err.Error())
}

// Error returns the error message.
func (e *gerror) Error() string {
	return e.msg
}

// String returns the msg as a string.
func (e *gerror) String() string {
	return e.msg
}

// Set the Error HTTP status code.
func (e *gerror) SetStatus(s int) {
	e.status = s
}

// Status returns the Error's HTTP status code.
func (e *gerror) Status() int {
	return e.status
}

// StatusError makes an error with a string and a HTTP status code.
func StatusError(msg string, status int) Error {
	e := &gerror{msg: msg, status: status}
	return e
}
