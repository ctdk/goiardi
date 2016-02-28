package search

import (
	"time"

	"github.com/ctdk/goiardi/config"
	"github.com/raintank/met"
	"github.com/tideland/golib/logger"
)

var inMemSearchTimings met.Timer
var pgSearchTimings met.Timer

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
