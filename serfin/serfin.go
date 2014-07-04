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
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/serf/serf"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var Serfer *serf.Serf

// StartSerfin sets up the serf instance and starts listening for events and
// queries from other serf instances.
func StartSerfin() error {
	events := make(chan serf.Event, 1)
	go func() {
		for {
			select {
				case event := <-events:
					logger.Debugf("Hey, got an event: %T %v", event, event)
			}
		}
	}()
	var mc *memberlist.Config
	switch config.Config.SerfNetType {
	case "lan":
		mc = memberlist.DefaultLANConfig()
	case "wan":
		mc = memberlist.DefaultWANConfig()
	case "local":
		mc = memberlist.DefaultLocalConfig()
	default:
		err := fmt.Errorf("'%s' is not a valid serf network type", config.Config.SerfNetType)
		return err
	}
	host, p, err := net.SplitHostPort(config.Config.SerfAddr)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return err
	}
	mc.BindAddr = host
	mc.BindPort = port
	// TODO: extend logger package to be able to return the log io.Writer
	// so the serf logs can go there
	serfConf := serf.DefaultConfig()
	serfConf.Init()
	serfConf.NodeName = config.Config.SerfNode
	// TODO: may want serf tags?
	serfConf.MemberlistConfig = mc
	serfConf.EventCh = events
	Serfer, err = serf.Create(serfConf)
	if err != nil {
		return err
	}
	joins := strings.Split(config.Config.SerfJoin, ",")
	if len(joins) != 0 {
		_, err = Serfer.Join(joins, false)
		if err != nil {
			logger.Warningf(err.Error())
		}
	}

	return nil
}
