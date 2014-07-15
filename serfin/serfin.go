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
	serfclient "github.com/hashicorp/serf/client"
	"os"
	"encoding/json"
	"time"
)

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
	if config.Config.ClusterName != "" {
		removeTags := []string{ "goiardi-cluster", "cluster-id" }
		err = Serfer.UpdateTags(nil, removeTags)
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		setTags := map[string]string{ "goiardi-cluster": config.Config.ClusterName, "cluster-id": config.Config.ClusterID }
		ack := make(chan string, 1)
		q := &serfclient.QueryParam{ Name: "helo", Payload: nil, RequestAck: true, FilterTags: setTags, AckCh: ack }
		err = Serfer.Query(q)
		if err != nil {
			logger.Criticalf("bwah: %s", err.Error())
		}
		select {
			case <-ack:
				logger.Criticalf("There is another goiardi node in cluster %s named %s", config.Config.ClusterName, config.Config.ClusterID)
				os.Exit(0)
			case <- time.After(500 * time.Millisecond):
				logger.Debugf("no duplicate goiardi nodes in the cluster")
		}
		
		err = Serfer.UpdateTags(setTags, nil)
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
	}
	
	err = Serfer.UserEvent("goiardi-join", []byte(config.Config.Hostname), true)
	if err != nil {
		logger.Criticalf(err.Error())
		os.Exit(1)
	}

	return nil
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
	q := &serfclient.QueryParam{ Name: queryName, Payload: jsonPayload }
	err = Serfer.Query(q)
	if err != nil {
		logger.Debugf(err.Error())
	}
	return
}
