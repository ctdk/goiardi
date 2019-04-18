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

// Package serfin bundles up serf functions for goiardi.
package serfin

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ctdk/goiardi/config"
	serfclient "github.com/hashicorp/serf/client"
	"github.com/tideland/golib/logger"
	"sync"
	"time"
)

// Serfer is the common serf client for goiardi. NB: moving away from doing it
// this way.
var Serfer *serfclient.RPCClient

type serfClientMap struct {
	serfs map[string]*serfclient.RPCClient
	m     sync.Mutex
}

var serfClients *serfClientMap

func init() {
	serfClients = new(serfClientMap)
	serfClients.serfs = make(map[string]*serfclient.RPCClient)
}

func (scm *serfClientMap) addSerfClient(sc *serfclient.RPCClient, serfAddr string) {
	scm.m.Lock()
	defer scm.m.Unlock()
	scm.serfs[serfAddr] = sc
	return
}

func (scm *serfClientMap) closeAllSerfs() {
	scm.m.Lock()
	defer scm.m.Unlock()
	for addr, sc := range scm.serfs {
		sc.Close()
		delete(scm.serfs, addr)
	}
}

func (scm *serfClientMap) closeSerf(serfAddr string) {
	scm.m.Lock()
	defer scm.m.Unlock()
	if sc, ok := scm.serfs[serfAddr]; ok {
		sc.Close()
		delete(scm.serfs, serfAddr)
	}
}

// StartSerfin sets up the serf instance and starts listening for events and
// queries from other serf instances.
func StartSerfin() error {
	var err error
	Serfer, err = NewRPCClient(config.Config.SerfAddr)
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

// Query makes a query to the default serf client, reconnecting if it's been
// closed.
func Query(q *serfclient.QueryParam, errch chan<- error) {
	var err error

	// retry connecting to serf for 5 minutes, every 5 seconds. TODO:
	// probably should make this configurable eventually.
	retryDelay := time.Duration(5)
	retryNum := 60

	if Serfer == nil || Serfer.IsClosed() {
		serfClients.closeSerf(config.Config.SerfAddr)
		var ns *serfclient.RPCClient
		for i := 0; i < retryNum; i++ {
			logger.Debugf("reconnecting to serf try #%d...", i+1)
			ns, err = NewRPCClient(config.Config.SerfAddr)
			if err == nil {
				Serfer = ns
				logger.Debugf("reconnected to serf!")
				break
			}
			logger.Debugf("Failed to reconnect to serf on try #%d, waiting %d seconds", i+1, retryDelay)
			time.Sleep(retryDelay * time.Second)
		}
		// if we got here we never managed to reconnect
		qErr := fmt.Errorf("Could not reconnect to serf after %d seconds. Last error: %s", int(retryDelay)*retryNum, err.Error())
		errch <- qErr
		close(errch)
		return
	}

	err = Serfer.Query(q)

	errch <- nil
	close(errch)
	return
}

func NewRPCClient(serfAddr string) (*serfclient.RPCClient, error) {
	sc, err := serfclient.NewRPCClient(serfAddr)
	if err != nil {
		return nil, err
	}
	serfClients.addSerfClient(sc, serfAddr)
	return sc, nil
}

// CloseAll closes all active serf clients
func CloseAll() {
	serfClients.closeAllSerfs()
}

// CloseSerfClient closes one serf client.
func CloseSerfClient(serfAddr string) {
	serfClients.closeSerf(serfAddr)
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
