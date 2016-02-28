/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jbingham@gmail.com>)
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
	"time"

	"github.com/ctdk/goiardi/config"
	"github.com/raintank/met"
	"github.com/tideland/golib/logger"
)

var inMemSearchTimings met.Timer
var pgSearchTimings met.Timer

// InitializeMetrics initializes the statsd timers for search queries.
func InitializeMetrics(metricsBackend met.Backend) {
	inMemSearchTimings = metricsBackend.NewTimer("search.in_mem", 0)
	pgSearchTimings = metricsBackend.NewTimer("search.pg", 0)
}

func trackSearchTiming(start time.Time, query string, timing met.Timer) {
	if !config.Config.UseStatsd {
		return
	}
	elapsed := time.Since(start)
	timing.Value(elapsed)
	logger.Debugf("search '%s' took %d microseconds", query, elapsed/time.Microsecond)
}
