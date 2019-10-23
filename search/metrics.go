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

// Functions and vars for search timing metrics in statsd.

package search

import (
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/organization"
	"github.com/raintank/met"
	"github.com/tideland/golib/logger"
	"strings"
	"sync"
	"time"
)

const (
	pgTiming    = "pg"
	inMemTiming = "in_mem"
)

type searchTimer struct {
	variety   string
	root      met.Timer
	orgTiming map[string]met.Timer
	backend   met.Backend
	m         *sync.Mutex
}

var searchTimings *searchTimer

// InitializeMetrics initializes the statsd timers for search queries.
func InitializeMetrics(metricsBackend met.Backend) {
	// harness the power of, um, the config.
	searchTimings = new(searchTimer)

	if config.Config.PgSearch {
		searchTimings.variety = pgTiming
	} else {
		searchTimings.variety = inMemTiming
	}

	searchTimings.backend = metricsBackend
	searchTimings.m = new(sync.Mutex)
	searchTimings.orgTiming = make(map[string]met.Timer)

	searchTimings.root = metricsBackend.NewTimer(searchTimings.rootMetricName(), 0)
}

func trackSearchTiming(org *organization.Organization, start time.Time, query string) {
	if !config.Config.UseStatsd {
		return
	}

	elapsed := time.Since(start)

	searchTimings.root.Value(elapsed)
	searchTimings.orgTime(org, elapsed)

	logger.Debugf("search '%s' in org '%s' took %d microseconds", query, org.Name, elapsed/time.Microsecond)
}

func (s *searchTimer) orgTime(org *organization.Organization, e time.Duration) {
	s.m.Lock()
	defer s.m.Unlock()

	if _, ok := s.orgTiming[org.Name]; !ok {
		s.orgTiming[org.Name] = s.backend.NewTimer(s.orgMetricName(org), 0)
	}
	s.orgTiming[org.Name].Value(e)
}

func (s *searchTimer) rootMetricName() string {
	return fmt.Sprintf("search.%s", s.variety)
}

func (s *searchTimer) orgMetricName(org *organization.Organization) string {
	r := s.rootMetricName()
	// strings.ToLower may be overkill?
	return fmt.Sprintf("%s.org.%s", r, strings.ToLower(org.Name))
}
