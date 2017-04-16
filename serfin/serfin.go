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

// Package serfin bundles up serf functions for goiardi.
package serfin

import (
	"encoding/json"
	"os"

	"github.com/ctdk/goiardi/config"
	serfclient "github.com/hashicorp/serf/client"
	"github.com/tideland/golib/logger"
)

// Serfer is the common serf client for goiardi.
var Serfer *serfclient.RPCClient

// StartSerfin sets up the serf instance and starts listening for events and
// queries from other serf instances.
func StartSerfin() error {
	var err error
	Serfer, err = serfclient.NewRPCClient(config.Config.SerfAddr)
	if err != nil {
		logger.Criticalf(err.Error())
		os.Exit(1)
	}

	if config.Config.SerfEventAnnounce {
		err = Serfer.UserEvent("goiardi-join", []byte(config.Config.Hostname), true)
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
	}

	return nil
}

// SendEvent sends a serf event out from goiardi.
func SendEvent(eventName string, payload interface{}) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf(err.Error())
		return
	}
	err = Serfer.UserEvent(eventName, jsonPayload, true)
	if err != nil {
		logger.Debugf(err.Error())
	}
	return
}

// SendQuery sends a basic, no frills query out over serf.
func SendQuery(queryName string, payload interface{}) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf(err.Error())
		return
	}
	q := &serfclient.QueryParam{Name: queryName, Payload: jsonPayload}
	err = Serfer.Query(q)
	if err != nil {
		logger.Debugf(err.Error())
	}
	return
}
