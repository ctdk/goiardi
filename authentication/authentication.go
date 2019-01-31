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

// Package authentication contains functions used to authenticate requests from
// the signed headers.
package authentication

/* Geez, import all the things why don't you. */

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/orgloader"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"

	"github.com/ctdk/chefcrypto"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CheckHeader checks the signed headers sent by the client against the expected
// result assembled from the request headers to verify their authorization.
func CheckHeader(userID string, r *http.Request) util.Gerror {
	pathArray := strings.Split(r.URL.Path[1:], "/")
	var org *organization.Organization
	if pathArray[0] == "organizations" && len(pathArray) > 1 {
		var err util.Gerror
		org, err = orgloader.Get(pathArray[1])
		if err != nil {
			return err
		}
	}
	u, err := actor.GetReqUser(org, userID)
	if err != nil {
		gerr := util.Errorf("Failed to authenticate as '%s'. Ensure that your node_name and client key are correct.", userID)
		gerr.SetStatus(http.StatusUnauthorized)
		return gerr
	}

	auerr := AuthenticateHeader(u.PublicKey(), config.Config.TimeSlewDur, r)
	if auerr != nil {
		return auerr
	}

	// check association last of all?
	if org != nil && u.IsUser() && !u.IsAdmin() {
		_, aerr := association.GetAssoc(u.(*user.User), org)
		if aerr != nil {
			return aerr
		}
	}
	return nil
}

// AuthenticateHeader authenticates the headers against the provided public key.
// In addition, this function supports providing a timeSlew, which is how much
// difference there can be between the host clock and the time in the request
// header.
func AuthenticateHeader(publicKey string, timeSlew time.Duration, r *http.Request) util.Gerror {
	// if timeSlew is zero use the default from the config package
	if timeSlew == time.Duration(0) {
		timeSlew, _ = time.ParseDuration(config.DefaultTimeSlew)
	}

	contentHash := r.Header.Get("X-OPS-CONTENT-HASH")
	if contentHash == "" {
		gerr := util.Errorf("no content hash provided")
		gerr.SetStatus(http.StatusBadRequest)
		return gerr
	}
	authTimestamp := r.Header.Get("x-ops-timestamp")
	if authTimestamp == "" {
		gerr := util.Errorf("no timestamp header provided")
		gerr.SetStatus(http.StatusBadRequest)
		return gerr
	}

	// check the time stamp w/ allowed slew
	tok, terr := checkTimeStamp(authTimestamp, timeSlew)
	if !tok {
		return terr
	}

	// Eventually this may be put to some sort of use, but for now just
	// make sure that it's there. Presumably eventually it would be used to
	// use algorithms other than sha1 for hashing the body, or using a
	// different version of the header signing algorithm.
	xopssign := r.Header.Get("x-ops-sign")
	var apiVer string
	if xopssign == "" {
		gerr := util.Errorf("missing X-Ops-Sign header")
		return gerr
	}
	re := regexp.MustCompile(`version=(\d+\.\d+)`)
	shaRe := regexp.MustCompile(`algorithm=(\w+)`)
	if verChk := re.FindStringSubmatch(xopssign); verChk != nil {
		apiVer = verChk[1]
		switch apiVer {
		case "1.0", "1.1", "1.2", "1.3":

		default:
			gerr := util.Errorf("Bad version number '%s' in X-Ops-Header", apiVer)
			return gerr
		}
	} else {
		gerr := util.Errorf("malformed version in X-Ops-Header")
		return gerr
	}

	// if algorithm is missing, it uses sha1. Of course, no other
	// hashing algorithm is supported yet...
	if shaChk := shaRe.FindStringSubmatch(xopssign); shaChk != nil {
		if shaChk[1] != "sha1" && !(shaChk[1] == "sha256" && apiVer == "1.3") {
			gerr := util.Errorf("Unsupported hashing algorithm '%s' specified in X-Ops-Header", shaChk[1])
			return gerr
		}
	}

	chkHash, chkerr := calcBodyHash(r, apiVer)
	if chkerr != nil {
		return chkerr
	}
	if chkHash != contentHash {
		gerr := util.Errorf("Content hash did not match hash of request body")
		gerr.SetStatus(http.StatusUnauthorized)
		return gerr
	}

	signedHeaders, sherr := assembleSignedHeader(r)
	if sherr != nil {
		return sherr
	}
	headToCheck := assembleHeaderToCheck(r, chkHash, apiVer)

	switch apiVer {
	case "1.3":
		chkerr = checkAuth13Headers(publicKey, r, headToCheck, signedHeaders)
	case "1.2":
		chkerr = checkAuth12Headers(publicKey, r, headToCheck, signedHeaders)
	default:
		chkerr = checkAuthHeaders(publicKey, r, headToCheck, signedHeaders)
	}

	if chkerr != nil {
		return chkerr
	}

	return nil
}

func checkAuth12Headers(publicKey string, r *http.Request, headToCheck, signedHeaders string) util.Gerror {
	sig, err := base64.StdEncoding.DecodeString(signedHeaders)
	if err != nil {
		gerr := util.CastErr(err)
		return gerr
	}
	sigSha := sha1.Sum([]byte(headToCheck))
	err = chefcrypto.Auth12HeaderVerify(publicKey, sigSha[:], sig)
	if err != nil {
		gerr := util.CastErr(err)
		gerr.SetStatus(http.StatusUnauthorized)
		return gerr
	}
	return nil
}

func checkAuth13Headers(publicKey string, r *http.Request, headToCheck, signedHeaders string) util.Gerror {
	sig, err := base64.StdEncoding.DecodeString(signedHeaders)
	if err != nil {
		gerr := util.CastErr(err)
		return gerr
	}
	sigSha := sha256.Sum256([]byte(headToCheck))
	err = chefcrypto.Auth13HeaderVerify(publicKey, sigSha[:], sig)
	if err != nil {
		return util.CastErr(err)
	}
	return nil
}

func checkAuthHeaders(publicKey string, r *http.Request, headToCheck, signedHeaders string) util.Gerror {
	decHead, berr := chefcrypto.HeaderDecrypt(publicKey, signedHeaders)

	if berr != nil {
		gerr := util.Errorf(berr.Error())
		gerr.SetStatus(http.StatusUnauthorized)
		return gerr
	}

	// debuggery
	log.Printf("decHead: ~~~\n%s\n~~~", decHead)
	log.Printf("headToCheck: ~~~\n%s~~~", headToCheck)
	if string(decHead) != headToCheck {
		gerr := util.Errorf("failed to verify authorization")
		gerr.SetStatus(http.StatusUnauthorized)
		return gerr
	}

	return nil
}

// liberated from net/http/httputil
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, nil, err
	}
	if err = b.Close(); err != nil {
		return nil, nil, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewBuffer(buf.Bytes())), nil
}

func assembleSignedHeader(r *http.Request) (string, util.Gerror) {
	sHeadStore := make(map[int]string)
	authHeader := regexp.MustCompile(`(?i)^X-Ops-Authorization-(\d+)`)
	for k := range r.Header {
		if c := authHeader.FindStringSubmatch(k); c != nil {
			/* Have to put it into a map first, then sort, in case
			 * the headers don't come out in the right order */
			// skipping this error because we shouldn't even be
			// able to get here with something that won't be an
			// integer. Famous last words, I'm sure.
			i, _ := strconv.Atoi(c[1])
			sHeadStore[i] = r.Header.Get(k)
		}
	}
	if len(sHeadStore) == 0 {
		gerr := util.Errorf("No authentication headers found!")
		gerr.SetStatus(http.StatusUnauthorized)
		return "", gerr
	}

	sH := make([]string, len(sHeadStore))
	sHlimit := len(sH)
	for k, v := range sHeadStore {
		if k > sHlimit {
			gerr := util.Errorf("malformed authentication headers")
			gerr.SetStatus(http.StatusUnauthorized)
			return "", gerr
		}
		sH[k-1] = v
	}
	signedHeaders := strings.Join(sH, "")

	return signedHeaders, nil
}

func checkTimeStamp(timestamp string, slew time.Duration) (bool, util.Gerror) {
	timeNow := time.Now().UTC()
	timeHeader, terr := time.Parse(time.RFC3339, timestamp)
	if terr != nil {
		err := util.Errorf(terr.Error())
		err.SetStatus(http.StatusUnauthorized)
		return false, err
	}
	tdiff := timeNow.Sub(timeHeader)
	// no easy integer based abs function
	if tdiff < 0 {
		tdiff = -tdiff
	}
	if tdiff > slew {
		err := util.Errorf("Authentication failed. Please check your system's clock.")
		err.SetStatus(http.StatusUnauthorized)
		return false, err
	}
	return true, nil
}

func assembleHeaderToCheck(r *http.Request, cHash string, apiVer string) string {
	method := r.Method
	cleanPath := path.Clean(r.URL.Path)
	timestamp := r.Header.Get("x-ops-timestamp")
	userID := r.Header.Get("x-ops-userid")
	if apiVer != "1.0" && apiVer != "1.3" {
		userID = hashStr(userID)
	}
	var headStr string
	if apiVer == "1.3" {
		headerXopsApiVersion := r.Header.Get("x-ops-server-api-version")
		headStr = fmt.Sprintf("Method:%s\nPath:%s\nX-Ops-Content-Hash:%s\nX-Ops-Sign:version=%s\nX-Ops-Timestamp:%s\nX-Ops-UserId:%s\nX-Ops-Server-API-Version:%s", method, cleanPath, cHash, apiVer, timestamp, userID, headerXopsApiVersion)
	} else {
		hashPath := hashStr(cleanPath)
		headStr = fmt.Sprintf("Method:%s\nHashed Path:%s\nX-Ops-Content-Hash:%s\nX-Ops-Timestamp:%s\nX-Ops-UserId:%s", method, hashPath, cHash, timestamp, userID)
	}

	return headStr
}

func hashStr(toHash string) string {
	h := sha1.New()
	return hashStrBase(toHash, h)
}

func hashSha256Str(toHash string) string {
	h := sha256.New()
	return hashStrBase(toHash, h)
}

func hashStrBase(toHash string, h hash.Hash) string {
	io.WriteString(h, toHash)
	hashed := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return hashed
}

func calcBodyHash(r *http.Request, apiVer string) (string, util.Gerror) {
	var bodyStr string
	if r.Body == nil {
		bodyStr = ""
	} else {
		var err error
		save := r.Body
		save, r.Body, err = drainBody(r.Body)
		if err != nil {
			gerr := util.Errorf("could not copy body")
			gerr.SetStatus(http.StatusInternalServerError)
			return "", gerr
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		bodyStr = buf.String()
		r.Body = save
	}
	var chkHash string
	if apiVer == "1.3" {
		chkHash = hashSha256Str(bodyStr)
	} else {
		chkHash = hashStr(bodyStr)
	}

	return chkHash, nil
}
