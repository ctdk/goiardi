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
	"golang.org/x/xerrors"
	"net/http"
)

// the private error struct
type gerror struct {
	err    error
	status int
}

// Error is an error type that includes an http status code (defaults to
// http.BadRequest).
type Error interface {
	String() string
	Error() string
	Status() int
	SetStatus(int)
	Unwrap() error
}

// New makes a new Error. Usually you want Errorf.
func New(text string) Error {
	return &gerror{err: xerrors.New(text),
		status: http.StatusBadRequest,
	}
}

// Errorf creates a new Error, with a formatted error string.
func Errorf(format string, a ...interface{}) Error {
	x := xerrors.Errorf(format, a...)
	return &gerror{err: x, status: http.StatusBadRequest}
}

// CastErr will easily cast a different kind of error to a goiardi Error.
func CastErr(err error) Error {
	var e *gerror

	// if the immediate error is a gerror, just return itself
	if !xerrors.As(err, &e) {
		e = &gerror{err: err, status: http.StatusBadRequest}
	}

	return e
}

// Error returns the error message.
func (e *gerror) Error() string {
	return e.err.Error()
}

// String returns the msg as a string.
func (e *gerror) String() string {
	// add the status maybe?
	return e.Error()
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
	e := &gerror{err: xerrors.New(msg), status: status}
	return e
}

// Unwrap implements the new Unwrap() interface for errors.
func (e *gerror) Unwrap() error {
	return e.err
}
