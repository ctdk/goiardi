/* A relatively simple Chef server implementation in Go, as a learning project
 * to learn more about programming in Go. */

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

package main

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/authentication"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/report"
	"github.com/ctdk/goiardi/reqctx"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/search"
	"github.com/ctdk/goiardi/secret"
	"github.com/ctdk/goiardi/serfin"
	"github.com/ctdk/goiardi/shovey"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	serfclient "github.com/hashicorp/serf/client"
	"github.com/raintank/met"
	"github.com/raintank/met/helper"
	"github.com/tideland/golib/logger"
)

type interceptHandler struct{} // Doesn't need to do anything, just sit there.

type apiTimerInfo struct {
	elapsed time.Duration
	path    string
	method  string
}

var noOpUserReqs = []string{
	"/authenticate_user",
	"/file_store",
	"/universe",
	"/principals",
	"/debug",
}

var apiChan chan *apiTimerInfo

func main() {
	config.ParseConfigOptions()

	/* Here goes nothing, db... */
	if config.UsingDB() {
		var derr error
		if config.Config.UseMySQL {
			datastore.Dbh, derr = datastore.ConnectDB("mysql", config.Config.MySQL)
		} else if config.Config.UsePostgreSQL {
			datastore.Dbh, derr = datastore.ConnectDB("postgres", config.Config.PostgreSQL)
		}
		if derr != nil {
			logger.Fatalf(derr.Error())
			os.Exit(1)
		}
	}

	// Set up secrets, if we're using them.
	if config.UsingExternalSecrets() {
		secerr := secret.ConfigureSecretStore()
		if secerr != nil {
			logger.Fatalf(secerr.Error())
			os.Exit(1)
		}
	}

	gobRegister()
	ds := datastore.New()
	indexer.Initialize(config.Config)
	if config.Config.FreezeData {
		if config.Config.DataStoreFile != "" {
			uerr := ds.Load(config.Config.DataStoreFile)
			if uerr != nil {
				logger.Fatalf(uerr.Error())
				os.Exit(1)
			}
		}
		ierr := indexer.LoadIndex()
		if ierr != nil {
			logger.Fatalf(ierr.Error())
			os.Exit(1)
		}
	}

	metricsBackend, merr := helper.New(config.Config.UseStatsd, config.Config.StatsdAddr, config.Config.StatsdType, "goiardi", config.Config.StatsdInstance)
	if merr != nil {
		logger.Fatalf(merr.Error())
		os.Exit(1)
	}
	util.InitS3(config.Config)
	initGeneralStatsd(metricsBackend)
	report.InitializeMetrics(metricsBackend)
	search.InitializeMetrics(metricsBackend)
	apiChan = make(chan *apiTimerInfo, 10) // unbuffered shouldn't block
	// anything, but a little buffer
	// shouldn't hurt
	go apiTimerMaster(apiChan, metricsBackend)

	setSaveTicker()
	setLogEventPurgeTicker()

	/* handle import/export */
	if config.Config.DoExport {
		fmt.Printf("Exporting data to %s....\n", config.Config.ImpExFile)
		err := exportAll(config.Config.ImpExFile)
		if err != nil {
			logger.Criticalf("Something went wrong during the export: %s", err.Error())
			os.Exit(1)
		}
		fmt.Println("All done!")
		os.Exit(0)
	} else if config.Config.DoImport {
		fmt.Printf("Importing data from %s....\n", config.Config.ImpExFile)
		err := importAll(config.Config.ImpExFile)
		if err != nil {
			logger.Criticalf("Something went wrong during the import: %s", err.Error())
			os.Exit(1)
		}
		if config.Config.FreezeData {
			if config.Config.DataStoreFile != "" {
				ds := datastore.New()
				if err := ds.Save(config.Config.DataStoreFile); err != nil {
					logger.Errorf(err.Error())
				}
			}
			if err := indexer.SaveIndex(); err != nil {
				logger.Errorf(err.Error())
			}
		}
		if config.UsingDB() {
			datastore.Dbh.Close()
		}
		fmt.Println("All done.")
		os.Exit(0)
	}

	/* Set up serf */
	if config.Config.UseSerf {
		serferr := serfin.StartSerfin()
		if serferr != nil {
			logger.Fatalf(serferr.Error())
			os.Exit(1)
		}
		errch := make(chan error)
		go startEventMonitor(config.Config.SerfAddr, errch)
		err := <-errch
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		startNodeMonitor()
	}

	if config.Config.PurgeNodeStatusAfter != "" {
		startNodeStatusPurge()
	}

	if config.Config.PurgeReportsAfter != "" {
		startReportPurge()
	}

	if config.Config.PurgeSandboxesDur != 0 {
		startSandboxPurge()
	}

	/* Create default clients and users. Currently chef-validator,
	 * chef-webui, and admin. */
	createDefaultActors()
	handleSignals()

	/* Register the various handlers, found in their own source files. */
	http.HandleFunc("/authenticate_user", authenticateUserHandler)
	http.HandleFunc("/clients", listHandler)
	http.HandleFunc("/clients/", clientHandler)
	http.HandleFunc("/cookbooks", cookbookHandler)
	http.HandleFunc("/cookbooks/", cookbookHandler)
	http.HandleFunc("/data", dataHandler)
	http.HandleFunc("/data/", dataHandler)
	http.HandleFunc("/environments", environmentHandler)
	http.HandleFunc("/environments/", environmentHandler)
	http.HandleFunc("/nodes", listHandler)
	http.HandleFunc("/nodes/", nodeHandler)
	http.HandleFunc("/principals/", principalHandler)
	http.HandleFunc("/roles", listHandler)
	http.HandleFunc("/roles/", roleHandler)
	http.HandleFunc("/sandboxes", sandboxHandler)
	http.HandleFunc("/sandboxes/", sandboxHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/search/", searchHandler)
	http.HandleFunc("/search/reindex", reindexHandler)
	http.HandleFunc("/users", listHandler)
	http.HandleFunc("/users/", userHandler)
	http.HandleFunc("/file_store/", fileStoreHandler)
	http.HandleFunc("/events", eventListHandler)
	http.HandleFunc("/events/", eventHandler)
	http.HandleFunc("/reports/", reportHandler)
	http.HandleFunc("/universe", universeHandler)
	http.HandleFunc("/shovey/", shoveyHandler)
	http.HandleFunc("/status/", statusHandler)

	/* TODO: figure out how to handle the root & not found pages */
	http.HandleFunc("/", rootHandler)

	listenAddr := config.ListenAddr()
	var err error
	srv := &http.Server{Addr: listenAddr, Handler: &interceptHandler{}}
	if config.Config.UseSSL {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS10}
		err = srv.ListenAndServeTLS(config.Config.SSLCert, config.Config.SSLKey)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		logger.Fatalf("ListenAndServe: %s", err.Error())
		os.Exit(1)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: make root do something useful
	return
}

func trackApiTiming(start time.Time, r *http.Request) {
	if !config.Config.UseStatsd {
		return
	}
	elapsed := time.Since(start)
	apiChan <- &apiTimerInfo{elapsed: elapsed, path: r.URL.Path, method: r.Method}
}

func apiTimerMaster(apiChan chan *apiTimerInfo, metricsBackend met.Backend) {
	if !config.Config.UseStatsd {
		return
	}
	metrics := make(map[string]met.Timer)
	for timeInfo := range apiChan {
		p := path.Clean(timeInfo.path)
		pathTmp := strings.Split(p, "/")
		if len(pathTmp) > 1 {
			p = pathTmp[1]
		} else {
			p = "root"
		}
		metricStr := fmt.Sprintf("api.timing.%s.%s", p, strings.ToLower(timeInfo.method))
		if _, ok := metrics[metricStr]; !ok {
			metrics[metricStr] = metricsBackend.NewTimer(metricStr, 0)
		}
		metrics[metricStr].Value(timeInfo.elapsed)

		logger.Debugf("in apiChan %s: %d microseconds %s %s", metricStr, timeInfo.elapsed/time.Microsecond, timeInfo.path, timeInfo.method)
	}
}

func (h *interceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/* knife sometimes sends URL paths that start with //. Redirecting
	 * worked for GETs, but since it was breaking POSTs and screwing with
	 * GETs with query params, we just clean up the path and move on. */

	// experimental - track time of api requests
	defer trackApiTiming(time.Now(), r)

	/* log the URL */
	// TODO: set this to verbosity level 4 or so
	logger.Debugf("Serving %s -- %s", r.URL.Path, r.Method)

	// block /debug/pprof if not localhost
	if strings.HasPrefix(r.URL.Path, "/debug") {
		fwded := strings.Split(r.Header.Get("X-Forwarded-For"), ", ")
		remoteIP, _, rerr := net.SplitHostPort(r.RemoteAddr)
		var block bool
		if rerr == nil {
			var xForwarded string
			if len(fwded) != 0 {
				xForwarded = fwded[len(fwded)-1]
			}
			rIP := net.ParseIP(remoteIP)
			xFIP := net.ParseIP(xForwarded)
			if !rIP.IsLoopback() && !xFIP.IsLoopback() && !config.PprofWhitelisted(rIP) && !config.PprofWhitelisted(xFIP) {
				logger.Debugf("blocked %s (x-forwarded-for: %s) from accessing /debug/pprof!", rIP.String(), xFIP.String())
				block = true
			}
		} else {
			logger.Debugf("remote ip %q is bad, not IP:port (blocking from /debug/pprof)", r.RemoteAddr)
			block = true
		}
		if block {
			http.Error(w, "Forbidden!", http.StatusForbidden)
			return
		}
	}

	if r.Method != "CONNECT" {
		if p := cleanPath(r.URL.Path); p != r.URL.Path {
			r.URL.Path = p
		}
	}

	/* Make configurable, I guess, but Chef wants it to be 1000000 */
	if !strings.HasPrefix(r.URL.Path, "/file_store") && r.ContentLength > config.Config.JSONReqMaxSize {
		logger.Debugf("Content length was too long for %s", r.URL.Path)
		http.Error(w, "Content-length too long!", http.StatusRequestEntityTooLarge)
		// hmm, with 1.5 it gets a broken pipe now if we don't do
		// anything with the body they're trying to send. Try copying it
		// to /dev/null. This seems crazy, but merely closing the body
		// doesn't actually work.
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
		return
	} else if r.ContentLength > config.Config.ObjMaxSize {
		http.Error(w, "Content-length waaaaaay too long!", http.StatusRequestEntityTooLarge)
		return
	}

	w.Header().Set("X-Goiardi", "yes")
	w.Header().Set("X-Goiardi-Version", config.Version)
	w.Header().Set("X-Chef-Version", config.ChefVersion)
	apiInfo := fmt.Sprintf("flavor=osc;version:%s;goiardi=%s", config.ChefVersion, config.Version)
	w.Header().Set("X-Ops-API-Info", apiInfo)

	apiver := r.Header.Get("X-Ops-Server-API-Version")
	if matchSupportedVersion(apiver) {
		w.Header().Set(
			"X-Ops-Server-API-Version",
			fmt.Sprintf(
				`{"min_version": config.MinAPIVersion ,"max_version": config.MaxAPIVersion, "request_version": "%s", "response_version": "%s"}`,
				apiver,
				apiver,
			),
		)
	}
	userID := r.Header.Get("X-OPS-USERID")
	if rs := r.Header.Get("X-Ops-Request-Source"); rs == "web" {
		/* If use-auth is on and disable-webui is on, and this is a
		 * webui connection, it needs to fail. */
		if config.Config.DisableWebUI {
			w.Header().Set("Content-Type", "application/json")
			logger.Warningf("Attempting to log in through webui, but webui is disabled")
			jsonErrorReport(w, r, "invalid action", http.StatusUnauthorized)
			return
		}

		/* Check that the user in question with the web request exists.
		 * If not, fail. */
		if _, uherr := actor.GetReqUser(userID); uherr != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Warningf("Attempting to use invalid user %s through X-Ops-Request-Source = web", userID)
			jsonErrorReport(w, r, "invalid action", http.StatusUnauthorized)
			return
		}
		userID = "chef-webui"
	}
	/* Only perform the authorization check if that's configured. Bomb with
	 * an error if the check of the headers, timestamps, etc. fails. */
	/* No clue why /principals doesn't require authorization. Hrmph. */
	if config.Config.UseAuth && !strings.HasPrefix(r.URL.Path, "/file_store") && !strings.HasPrefix(r.URL.Path, "/debug") && !(strings.HasPrefix(r.URL.Path, "/principals") && r.Method == "GET") {
		herr := authentication.CheckHeader(userID, r)
		if herr != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Errorf("Authorization failure: %s\n", herr.Error())
			w.Header().Set("Www-Authenticate", `X-Ops-Sign version="1.0" version="1.1" version="1.2" version="1.3"`)
			jsonErrorReport(w, r, herr.Error(), herr.Status())
			return
		}
	}

	// Experimental: decompress gzipped requests
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Errorf("Failure decompressing gzipped request body: %s\n", err.Error())
			jsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = reader
	}

	// Set up the context for the request. At this time, this means setting
	// the opUser for this request for most (but not all) types of requests.
	// At this time the exceptions are "/file_store", "/universe", and
	// "/authenticate_user".
	ctx := r.Context()
	var skip bool
	for _, p := range noOpUserReqs {
		if strings.HasPrefix(r.URL.Path, p) {
			skip = true
			break
		}
	}
	if !skip {
		opUser, oerr := actor.GetReqUser(r.Header.Get("X-OPS-USERID"))
		if oerr != nil {
			w.Header().Set("Content-Type", "application/json")
			jsonErrorReport(w, r, oerr.Error(), oerr.Status())
			return
		}
		ctx = context.WithValue(ctx, reqctx.OpUserKey, opUser)
	}

	http.DefaultServeMux.ServeHTTP(w, r.WithContext(ctx))
}

func cleanPath(p string) string {
	/* Borrowing cleanPath from net/http */
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

func createDefaultActors() {
	if cwebui, _ := client.Get("chef-webui"); cwebui == nil {
		if webui, nerr := client.New("chef-webui"); nerr != nil {
			logger.Criticalf(nerr.Error())
			os.Exit(1)
		} else {
			webui.Admin = true
			pem, err := webui.GenerateKeys()
			if err != nil {
				logger.Criticalf(err.Error())
				os.Exit(1)
			}
			if config.Config.UseAuth {
				if fp, ferr := os.Create(fmt.Sprintf("%s/%s.pem", config.Config.ConfRoot, webui.Name)); ferr == nil {
					fp.Chmod(0600)
					fp.WriteString(pem)
					fp.Close()
				} else {
					logger.Criticalf(ferr.Error())
					os.Exit(1)
				}
			}

			webui.Save()
		}
	}

	if cvalid, _ := client.Get("chef-validator"); cvalid == nil {
		if validator, verr := client.New("chef-validator"); verr != nil {
			logger.Criticalf(verr.Error())
			os.Exit(1)
		} else {
			validator.Validator = true
			pem, err := validator.GenerateKeys()
			if err != nil {
				logger.Criticalf(err.Error())
				os.Exit(1)
			}
			if config.Config.UseAuth {
				if fp, ferr := os.Create(fmt.Sprintf("%s/%s.pem", config.Config.ConfRoot, validator.Name)); ferr == nil {
					fp.Chmod(0600)
					fp.WriteString(pem)
					fp.Close()
				} else {
					logger.Criticalf(ferr.Error())
					os.Exit(1)
				}
			}
			validator.Save()
		}
	}

	if uadmin, _ := user.Get("admin"); uadmin == nil {
		if admin, aerr := user.New("admin"); aerr != nil {
			logger.Criticalf(aerr.Error())
			os.Exit(1)
		} else {
			admin.Admin = true
			pem, err := admin.GenerateKeys()
			if err != nil {
				logger.Criticalf(err.Error())
				os.Exit(1)
			}
			if config.Config.UseAuth {
				if fp, ferr := os.Create(fmt.Sprintf("%s/%s.pem", config.Config.ConfRoot, admin.Name)); ferr == nil {
					fp.Chmod(0600)
					fp.WriteString(pem)
					fp.Close()
				} else {
					logger.Criticalf(ferr.Error())
					os.Exit(1)
				}
			}
			if aerr := admin.Save(); aerr != nil {
				logger.Criticalf(aerr.Error())
				os.Exit(1)
			}
		}
	}

	environment.MakeDefaultEnvironment()

	return
}

func handleSignals() {
	c := make(chan os.Signal, 1)
	// SIGTERM is not exactly portable, but Go has a fake signal for it
	// with Windows so it being there should theoretically not break it
	// running on windows
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	// if we receive a SIGINT or SIGTERM, do cleanup here.
	go func() {
		for sig := range c {
			if sig == os.Interrupt || sig == syscall.SIGTERM {
				logger.Infof("cleaning up...")
				if config.Config.FreezeData {
					if config.Config.DataStoreFile != "" {
						ds := datastore.New()
						if err := ds.Save(config.Config.DataStoreFile); err != nil {
							logger.Errorf(err.Error())
						}
					}
					if err := indexer.SaveIndex(); err != nil {
						logger.Errorf(err.Error())
					}
				}
				if config.UsingDB() {
					datastore.Dbh.Close()
				}
				if config.Config.UseSerf {
					serfin.CloseAll()
				}
				os.Exit(0)
			} else if sig == syscall.SIGHUP {
				logger.Infof("Reloading configuration...")
				config.ParseConfigOptions()
			}
		}
	}()
}

func gobRegister() {
	e := new(environment.ChefEnvironment)
	gob.Register(e)
	c := new(cookbook.Cookbook)
	gob.Register(c)
	d := new(databag.DataBag)
	gob.Register(d)
	f := new(filestore.FileStore)
	gob.Register(f)
	n := new(node.Node)
	gob.Register(n)
	r := new(role.Role)
	gob.Register(r)
	s := new(sandbox.Sandbox)
	gob.Register(s)
	m := make(map[string]interface{})
	gob.Register(m)
	var si []interface{}
	gob.Register(si)
	var ss []string
	gob.Register(ss)
	ms := make(map[string]string)
	gob.Register(ms)
	var smsi []map[string]interface{}
	gob.Register(smsi)
	msss := make(map[string][]string)
	gob.Register(msss)
	cc := new(client.Client)
	gob.Register(cc)
	uu := new(user.User)
	gob.Register(uu)
	li := new(loginfo.LogInfo)
	gob.Register(li)
	mis := map[int]interface{}{}
	gob.Register(mis)
	cbv := new(cookbook.CookbookVersion)
	gob.Register(cbv)
	dbi := new(databag.DataBagItem)
	gob.Register(dbi)
	rp := new(report.Report)
	gob.Register(rp)
	sv := new(shovey.Shovey)
	gob.Register(sv)
	svr := new(shovey.ShoveyRun)
	gob.Register(svr)
	svs := new(shovey.ShoveyRunStream)
	gob.Register(svs)
	ns := new(node.NodeStatus)
	gob.Register(ns)
	msi := make(map[string][]int)
	gob.Register(msi)
	var jn json.Number
	gob.Register(jn)
}

func setSaveTicker() {
	if config.Config.FreezeData {
		ds := datastore.New()
		ticker := time.NewTicker(time.Second * time.Duration(config.Config.FreezeInterval))
		go func() {
			for _ = range ticker.C {
				if config.Config.DataStoreFile != "" {
					uerr := ds.Save(config.Config.DataStoreFile)
					if uerr != nil {
						logger.Errorf(uerr.Error())
					}
				}
				ierr := indexer.SaveIndex()
				if ierr != nil {
					logger.Errorf(ierr.Error())
				}
			}
		}()
	}
}

func setLogEventPurgeTicker() {
	if config.Config.LogEventKeep != 0 {
		ticker := time.NewTicker(time.Second * time.Duration(60))
		go func() {
			for _ = range ticker.C {
				les, _ := loginfo.GetLogInfos(nil, 0, 1)
				if len(les) != 0 {
					p, err := loginfo.PurgeLogInfos(les[0].ID - config.Config.LogEventKeep)
					if err != nil {
						logger.Errorf(err.Error())
					}
					logger.Debugf("Purged %d events automatically", p)
				}
			}
		}()
	}
}

// The serf functionality needs some cleaning up. This is a start on that.
func startEventMonitor(serfAddr string, errch chan<- error) {
	// Initial setup of serf. If this bombs go ahead and return so we can
	// die
	sc, err := serfin.NewRPCClient(serfAddr)
	if err != nil {
		errch <- err
		return
	}
	errch <- nil

	ech := make(chan error)
	recreateSerfWait := time.Duration(5)

	for {
		// Make sure the serf client is actually closed before creating
		// a new one. The very first time this loop is kicked off, of
		// course, the client will be fine. It's simpler to have the
		// check up here, though, rather than at the end
		if sc == nil || sc.IsClosed() {
			sc, err = serfin.NewRPCClient(serfAddr)
			if err != nil {
				logger.Errorf("Error recreating serf client, waiting %d seconds before recreating: %s", recreateSerfWait, err.Error())
				time.Sleep(recreateSerfWait * time.Second)
				continue
			} else {
				logger.Errorf("reconnected to serf after being disconnected")
			}
		}
		go runEventMonitor(sc, ech)
		e := <-ech
		if e != nil {
			logger.Errorf("Error from event monitor: %s", e.Error())
		}
	}
}

func runEventMonitor(sc *serfclient.RPCClient, errch chan<- error) {
	ch := make(chan map[string]interface{}, 10)
	sh, err := sc.Stream("*", ch)
	if err != nil {
		errch <- err
		return
	}

	defer sc.Stop(sh)
	checkClientSec := time.Duration(15)

	// watch the events and queries
	for {
		select {
		case e := <-ch:
			eNil := e == nil
			logger.Debugf("Got an event: %v nil? %v", e, eNil)
			if eNil {
				if sc.IsClosed() {
					logger.Debugf("Serf client has been closed, returning from runEventMonitor in hopes of being able to reconnect")
					err := fmt.Errorf("serf client closed")
					errch <- err
					return
				}
				continue
			}
			eName, _ := e["Name"]
			switch eName {
			case "node_status":
				jsonPayload := make(map[string]string)
				err = json.Unmarshal(e["Payload"].([]byte), &jsonPayload)
				if err != nil {
					logger.Errorf(err.Error())
					continue
				}
				n, _ := node.Get(jsonPayload["node"])
				if n == nil {
					logger.Errorf("No node %s", jsonPayload["node"])
					continue
				}
				err = n.UpdateStatus(jsonPayload["status"])
				if err != nil {
					logger.Errorf(err.Error())
					continue
				}
				r := map[string]string{"response": "ok"}
				response, _ := json.Marshal(r)
				var id uint64
				switch t := e["ID"].(type) {
				case int64:
					id = uint64(t)
				case uint64:
					id = t
				default:
					logger.Errorf("node_status ID %v type %T not int64 or uint64", e["ID"], e["ID"])
					continue
				}
				sc.Respond(id, response)
			}
		case <-time.After(checkClientSec * time.Second):
			if sc.IsClosed() {
				clerr := fmt.Errorf("serf client found to be closed, recreating")
				errch <- clerr
				return
			}
		}
	}
}

func startNodeMonitor() {
	// Never do this if serf isn't set up
	if !config.Config.UseSerf {
		return
	}
	go func() {
		// wait 1 minute before starting to check for nodes being up
		time.Sleep(1 * time.Minute)
		ticker := time.NewTicker(time.Minute)
		for _ = range ticker.C {
			unseen, err := node.UnseenNodes()
			if err != nil {
				logger.Errorf(err.Error())
				continue
			}
			for _, n := range unseen {
				logger.Infof("Haven't seen %s for a while, marking as down", n.Name)
				err = n.UpdateStatus("down")
				if err != nil {
					logger.Errorf(err.Error())
					continue
				}
			}
		}
	}()
	return
}

func startReportPurge() {
	go func() {
		// purge reports after 2 hours, I guess.
		ticker := time.NewTicker(2 * time.Hour)
		for _ = range ticker.C {
			del, err := report.DeleteByAge(config.Config.PurgeReportsDur)
			if err != nil {
				logger.Errorf("Purging reports had an error: %s", err.Error())
			} else {
				logger.Debugf("Purged %d reports", del)
			}
		}
	}()
}

func startSandboxPurge() {
	go func() {
		// check for sandboxes to purge every hour
		ticker := time.NewTicker(time.Hour)
		for _ = range ticker.C {
			del, err := sandbox.Purge(config.Config.PurgeSandboxesDur)
			if err != nil {
				logger.Errorf("Purging sandboxes (somehow) had an error: %s", err.Error())
			} else {
				logger.Debugf("Purged %d sandboxes", del)
			}
		}
	}()
}

func startNodeStatusPurge() {
	// don't do it if there aren't going to be node statuses to purge
	if !config.Config.UseSerf || config.Config.PurgeNodeStatusDur == 0 {
		return
	}
	go func() {
		// check every 2 hours for statuses to purge
		ticker := time.NewTicker(2 * time.Hour)
		for _ = range ticker.C {
			del, err := node.DeleteNodeStatusesByAge(config.Config.PurgeNodeStatusDur)
			if err != nil {
				logger.Errorf("Purging node statuses had an error: %s", err.Error())
			} else {
				logger.Debugf("Purged %d node statuses", del)
			}
		}
	}()
}

func initGeneralStatsd(metricsBackend met.Backend) {
	if !config.Config.UseStatsd {
		return
	}
	// a count of the nodes on this server. Add other gauges later, but
	// start with this one.
	nodeCountGauge := metricsBackend.NewGauge("node.count", node.Count())

	// Taking some inspiration from this page I found:
	// http://zqpythonic.qiniucdn.com/data/20131112090955/index.html
	// -- which does not seem to be the original source of it, but that
	// seems to be gone -- we'll also take metrics of the golang runtime.
	memStats := &runtime.MemStats{}
	// initial reading in of memstat data
	runtime.ReadMemStats(memStats)
	lastSampleTime := time.Now()

	numGoroutine := metricsBackend.NewGauge("runtime.goroutines", int64(runtime.NumGoroutine()))
	allocated := metricsBackend.NewGauge("runtime.memory.allocated", int64(memStats.Alloc))
	mallocs := metricsBackend.NewGauge("runtime.memory.mallocs", int64(memStats.Mallocs))
	frees := metricsBackend.NewGauge("runtime.memory.frees", int64(memStats.Frees))
	totalPause := metricsBackend.NewGauge("runtime.gc.total_pause", int64(memStats.PauseTotalNs))
	heapAlloc := metricsBackend.NewGauge("runtime.memory.heap", int64(memStats.HeapAlloc))
	stackInUse := metricsBackend.NewGauge("runtime.memory.stack", int64(memStats.StackInuse))
	pausePerSec := metricsBackend.NewGauge("runtime.gc.pause_per_sec", 0)
	pausePerTick := metricsBackend.NewGauge("runtime.gc.pause_per_tick", 0)
	numGCTotal := metricsBackend.NewGauge("runtime.gc.num_gc", int64(memStats.NumGC))
	gcPerSec := metricsBackend.NewGauge("runtime.gc.gc_per_sec", 0)
	gcPerTick := metricsBackend.NewGauge("runtime.gc.gc_per_tick", 0)
	gcPause := metricsBackend.NewTimer("runtime.gc.pause", 0)

	lastPause := memStats.PauseTotalNs
	lastGC := memStats.NumGC

	statsdTickInt := 10

	// update the gauges every 10 seconds. Make this configurable later?
	go func() {
		ticker := time.NewTicker(time.Duration(statsdTickInt) * time.Second)
		for _ = range ticker.C {
			runtime.ReadMemStats(memStats)
			now := time.Now()

			nodeCountGauge.Value(node.Count())
			numGoroutine.Value(int64(runtime.NumGoroutine()))
			allocated.Value(int64(memStats.Alloc))
			mallocs.Value(int64(memStats.Mallocs))
			frees.Value(int64(memStats.Frees))
			totalPause.Value(int64(memStats.PauseTotalNs))
			heapAlloc.Value(int64(memStats.HeapAlloc))
			stackInUse.Value(int64(memStats.StackInuse))
			numGCTotal.Value(int64(memStats.NumGC))

			p := int(memStats.PauseTotalNs - lastPause)
			pausePerSec.Value(int64(p / statsdTickInt))
			pausePerTick.Value(int64(p))

			countGC := int64(memStats.NumGC - lastGC)
			diffTime := int64(now.Sub(lastSampleTime).Seconds())
			gcPerSec.Value(countGC / diffTime)
			gcPerTick.Value(countGC)

			if countGC > 0 {
				if countGC > 256 {
					logger.Warningf("lost some gc pause times")
					countGC = 256
				}
				var i int64
				for i = 0; i < countGC; i++ {
					idx := int((memStats.NumGC-uint32(i))+255) % 256
					pause := time.Duration(memStats.PauseNs[idx])
					gcPause.Value(pause)
				}
			}

			lastPause = memStats.PauseTotalNs
			lastGC = memStats.NumGC
			lastSampleTime = now
		}
	}()
}

func matchSupportedVersion(ver string) bool {
	for _, v := range config.SupportedAPIVersions {
		if ver == v {
			return true
		}
	}
	return false
}
