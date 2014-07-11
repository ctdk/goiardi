/* A relatively simple Chef server implementation in Go, as a learning project
 * to learn more about programming in Go. */

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

package main

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"github.com/ctdk/goas/v2/logger"
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
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/serfin"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
)

type interceptHandler struct{} // Doesn't need to do anything, just sit there.

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
			logger.Criticalf(derr.Error())
			os.Exit(1)
		}
	}

	gobRegister()
	ds := datastore.New()
	if config.Config.FreezeData {
		if config.Config.DataStoreFile != "" {
			uerr := ds.Load(config.Config.DataStoreFile)
			if uerr != nil {
				logger.Criticalf(uerr.Error())
				os.Exit(1)
			}
		}
		ierr := indexer.LoadIndex(config.Config.IndexFile)
		if ierr != nil {
			logger.Criticalf(ierr.Error())
			os.Exit(1)
		}
	}
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
			if err := indexer.SaveIndex(config.Config.IndexFile); err != nil {
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
			logger.Criticalf(serferr.Error())
			os.Exit(1)
		}
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
	if config.Config.UseSSL {
		err = http.ListenAndServeTLS(listenAddr, config.Config.SSLCert, config.Config.SSLKey, &interceptHandler{})
	} else {
		err = http.ListenAndServe(listenAddr, &interceptHandler{})
	}
	if err != nil {
		logger.Criticalf("ListenAndServe: %s", err.Error())
		os.Exit(1)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: make root do something useful
	return
}

func (h *interceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/* knife sometimes sends URL paths that start with //. Redirecting
	 * worked for GETs, but since it was breaking POSTs and screwing with
	 * GETs with query params, we just clean up the path and move on. */

	/* log the URL */
	// TODO: set this to verbosity level 4 or so
	logger.Debugf("Serving %s -- %s", r.URL.Path, r.Method)

	if r.Method != "CONNECT" {
		if p := cleanPath(r.URL.Path); p != r.URL.Path {
			r.URL.Path = p
		}
	}

	/* Make configurable, I guess, but Chef wants it to be 1000000 */
	if !strings.HasPrefix(r.URL.Path, "/file_store") && r.ContentLength > config.Config.JSONReqMaxSize {
		http.Error(w, "Content-length too long!", http.StatusRequestEntityTooLarge)
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
	if config.Config.UseAuth && !strings.HasPrefix(r.URL.Path, "/file_store") && !(strings.HasPrefix(r.URL.Path, "/principals") && r.Method == "GET") {
		herr := authentication.CheckHeader(userID, r)
		if herr != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Errorf("Authorization failure: %s\n", herr.Error())
			w.Header().Set("Www-Authenticate", `X-Ops-Sign version="1.0" version="1.1" version="1.2"`)
			//http.Error(w, herr.Error(), herr.Status())
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

	http.DefaultServeMux.ServeHTTP(w, r)
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
					if err := indexer.SaveIndex(config.Config.IndexFile); err != nil {
						logger.Errorf(err.Error())
					}
				}
				if config.UsingDB() {
					datastore.Dbh.Close()
				}
				if config.Config.UseSerf {
					serfin.Serfer.Close()
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
	i := new(indexer.Index)
	ic := new(indexer.IdxCollection)
	id := new(indexer.IdxDoc)
	gob.Register(i)
	gob.Register(ic)
	gob.Register(id)
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
}

func setSaveTicker() {
	if config.Config.FreezeData {
		ds := datastore.New()
		ticker := time.NewTicker(time.Second * time.Duration(config.Config.FreezeInterval))
		go func() {
			for _ = range ticker.C {
				if config.Config.DataStoreFile != "" {
					logger.Infof("Automatically saving data store...")
					uerr := ds.Save(config.Config.DataStoreFile)
					if uerr != nil {
						logger.Errorf(uerr.Error())
					}
				}
				logger.Infof("Automatically saving index...")
				ierr := indexer.SaveIndex(config.Config.IndexFile)
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
