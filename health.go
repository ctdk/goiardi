package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ctdk/goiardi/user"

	"github.com/ctdk/goiardi/util"

	"github.com/ctdk/goiardi/config"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type typeHealthCheckItem struct {
		Ok     bool   `json:"ok"`
		TimeMs int64  `json:"time_ms"`
		Error  string `json:"error,omitempty"`
	}
	type typeHealthCheckResult struct {
		Ok bool                 `json:"ok"`
		DB *typeHealthCheckItem `json:"db,omitempty"`
		S3 *typeHealthCheckItem `json:"s3,omitempty"`
	}
	//
	healthCheckResult := typeHealthCheckResult{
		Ok: true,
	}

	//s3 check
	if config.Config.UseS3Upload {
		start := time.Now()
		res, err := util.S3HealthCheck()
		healthCheckResult.S3 = &typeHealthCheckItem{
			Ok:     res,
			TimeMs: time.Since(start).Microseconds(),
		}
		if err != nil {
			healthCheckResult.Ok = false
			healthCheckResult.S3.Error = err.Error()
		}
	}

	//db check
	if config.UsingDB() {
		start := time.Now()
		//since we are using db, try to fetch user admin which always exists. if there is no error, then db is fine.
		_, err := user.Get("admin")
		healthCheckResult.DB = &typeHealthCheckItem{
			Ok:     true,
			TimeMs: time.Since(start).Microseconds(),
		}
		if err != nil {
			healthCheckResult.Ok = false
			healthCheckResult.DB.Ok = false
			healthCheckResult.DB.Error = err.Error()
		}
	}

	jsonResult, err := json.Marshal(&healthCheckResult)
	if err != nil {
		jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	switch healthCheckResult.Ok {
	case true:
		w.WriteHeader(http.StatusOK)
	case false:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_, _ = w.Write(jsonResult)
}
