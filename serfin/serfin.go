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

// Package serfin 
package serfin

import (
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/config"
	"github.com/hashicorp/serf/client"
	"os"
	"encoding/json"
)

var Serfer *client.RPCClient

// StartSerfin sets up the serf instance and starts listening for events and
// queries from other serf instances.
func StartSerfin() error {
	var err error
	Serfer, err = client.NewRPCClient("127.0.0.1:7373")
	if err != nil {
		logger.Criticalf(err.Error())
		os.Exit(1)
	}
	err = Serfer.UserEvent("goiardi-join", []byte(config.Config.Hostname), true)
	if err != nil {
		logger.Criticalf(err.Error())
		os.Exit(1)
	}
	errch := make(chan error, 1)
	go startEventMonitor(Serfer, errch)

	err = <-errch
	if err != nil {
		logger.Errorf(err.Error())
		os.Exit(1)
	}

	return nil
}

func startEventMonitor(sc *client.RPCClient, errch chan<- error) {
	ch := make(chan map[string]interface{}, 1)
	sh, err := sc.Stream("*", ch)
	if err != nil {
		errch <- err
		return
	}
	errch <- nil

	defer sc.Stop(sh)
	// watch the events and queries
	for e := range ch {
		logger.Debugf("Got an event: %v", e)
	}
	return
}

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

func SendQuery(queryName string, payload interface{}) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Errorf(err.Error())
		return
	}
	q := &client.QueryParam{ Name: queryName, Payload: jsonPayload }
	err = Serfer.Query(q)
	if err != nil {
		logger.Debugf(err.Error())
	}
	return
}
