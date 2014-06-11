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
	"net/http"
	"path"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/data_store"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/data_bag"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/log_info"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"encoding/gob"
	"time"
	"github.com/ctdk/goiardi/authentication"
	"strings"
	"git.tideland.biz/goas/logger"
	"compress/gzip"
)

type InterceptHandler struct {} // Doesn't need to do anything, just sit there.

func main(){
	config.ParseConfigOptions()

	/* Here goes nothing, db... */
	if config.Config.UseMySQL {
		var derr error
		data_store.Dbh, derr = data_store.ConnectDB("mysql", config.Config.MySQL)
		if derr != nil {
			logger.Criticalf(derr.Error())
			os.Exit(1)
		}
	}

	gobRegister()
	ds := data_store.New()
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
				ds := data_store.New()
				if err := ds.Save(config.Config.DataStoreFile); err != nil {
					logger.Errorf(err.Error())
				}
			}
			if err := indexer.SaveIndex(config.Config.IndexFile); err != nil {
				logger.Errorf(err.Error())
			}
		}
		if config.Config.UseMySQL {
			data_store.Dbh.Close()
		}
		fmt.Println("All done.")
		os.Exit(0)
	}

	/* Create default clients and users. Currently chef-validator,
	 * chef-webui, and admin. */
	createDefaultActors()
	handleSignals()

	/* Register the various handlers, found in their own source files. */
	http.HandleFunc("/authenticate_user", authenticate_user_handler)
	http.HandleFunc("/clients", list_handler)
	http.HandleFunc("/clients/", client_handler)
	http.HandleFunc("/cookbooks", cookbook_handler)
	http.HandleFunc("/cookbooks/", cookbook_handler)
	http.HandleFunc("/data", data_handler)
	http.HandleFunc("/data/", data_handler)
	http.HandleFunc("/environments", environment_handler)
	http.HandleFunc("/environments/", environment_handler)
	http.HandleFunc("/nodes", list_handler)
	http.HandleFunc("/nodes/", node_handler)
	http.HandleFunc("/principals/", principal_handler)
	http.HandleFunc("/roles", list_handler)
	http.HandleFunc("/roles/", role_handler)
	http.HandleFunc("/sandboxes", sandbox_handler)
	http.HandleFunc("/sandboxes/", sandbox_handler)
	http.HandleFunc("/search", search_handler)
	http.HandleFunc("/search/", search_handler)
	http.HandleFunc("/search/reindex", reindexHandler)
	http.HandleFunc("/users", list_handler)
	http.HandleFunc("/users/", user_handler)
	http.HandleFunc("/file_store/", file_store_handler)
	http.HandleFunc("/events", event_list_handler)
	http.HandleFunc("/events/", event_handler)
	http.HandleFunc("/reports/", report_handler)

	/* TODO: figure out how to handle the root & not found pages */
	http.HandleFunc("/", root_handler)

	listen_addr := config.ListenAddr()
	var err error
	if config.Config.UseSSL {
		err = http.ListenAndServeTLS(listen_addr, config.Config.SslCert, config.Config.SslKey, &InterceptHandler{})
	} else {
		err = http.ListenAndServe(listen_addr, &InterceptHandler{})
	}
	if err != nil {
		logger.Criticalf("ListenAndServe: %s", err.Error())
		os.Exit(1)
	}
}

func root_handler(w http.ResponseWriter, r *http.Request){
	// TODO: make root do something useful
	return
}

func (h *InterceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request){
	/* knife sometimes sends URL paths that start with //. Redirecting
	 * worked for GETs, but since it was breaking POSTs and screwing with 
	 * GETs with query params, we just clean up the path and move on. */

	/* log the URL */
	// TODO: set this to verbosity level 4 or so
	logger.Debugf("Serving %s -- %s\n", r.URL.Path, r.Method)

	if r.Method != "CONNECT" { 
		if p := cleanPath(r.URL.Path); p != r.URL.Path{
			r.URL.Path = p
		}
	}

	/* Make configurable, I guess, but Chef wants it to be 1000000 */
	if r.ContentLength > 1000000 {
		http.Error(w, "Content-length too long!", http.StatusRequestEntityTooLarge)
		return
	}

	w.Header().Set("X-Goiardi", "yes")
	w.Header().Set("X-Goiardi-Version", config.Version)
	w.Header().Set("X-Chef-Version", config.ChefVersion)
	api_info := fmt.Sprintf("flavor=osc;version:%s;goiardi=%s", config.ChefVersion, config.Version)
	w.Header().Set("X-Ops-API-Info", api_info)

	user_id := r.Header.Get("X-OPS-USERID")
	if rs := r.Header.Get("X-Ops-Request-Source"); rs == "web" {
		/* If use-auth is on and disable-webui is on, and this is a
		 * webui connection, it needs to fail. */
		if config.Config.DisableWebUI {
			w.Header().Set("Content-Type", "application/json")
			logger.Warningf("Attempting to log in through webui, but webui is disabled")
			JsonErrorReport(w, r, "invalid action", http.StatusUnauthorized)
			return
		}

		/* Check that the user in question with the web request exists.
		 * If not, fail. */
		if _, uherr := actor.GetReqUser(user_id); uherr != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Warningf("Attempting to use invalid user %s through X-Ops-Request-Source = web", user_id)
			JsonErrorReport(w, r, "invalid action", http.StatusUnauthorized)
			return
		}
		user_id = "chef-webui"
	}
	/* Only perform the authorization check if that's configured. Bomb with
	 * an error if the check of the headers, timestamps, etc. fails. */
	/* No clue why /principals doesn't require authorization. Hrmph. */
	if config.Config.UseAuth && !strings.HasPrefix(r.URL.Path, "/file_store") && !(strings.HasPrefix(r.URL.Path, "/principals") && r.Method == "GET") {
		herr := authentication.CheckHeader(user_id, r)
		if herr != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Errorf("Authorization failure: %s\n", herr.Error())
			//http.Error(w, herr.Error(), herr.Status())
			JsonErrorReport(w, r, herr.Error(), herr.Status())
			return
		}
	}

	// Experimental: decompress gzipped requests
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Errorf("Failure decompressing gzipped request body: %s\n", err.Error())
			JsonErrorReport(w, r, err.Error(), http.StatusBadRequest)
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
			admin.Save()
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
	go func(){
		for sig := range c {
			if sig == os.Interrupt || sig == syscall.SIGTERM{
				logger.Infof("cleaning up...")
				if config.Config.FreezeData {
					if config.Config.DataStoreFile != "" {
						ds := data_store.New()
						if err := ds.Save(config.Config.DataStoreFile); err != nil {
							logger.Errorf(err.Error())
						}
					}
					if err := indexer.SaveIndex(config.Config.IndexFile); err != nil {
						logger.Errorf(err.Error())
					}
				}
				if config.Config.UseMySQL {
					data_store.Dbh.Close()
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
	d := new(data_bag.DataBag)
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
	si := make([]interface{},0)
	gob.Register(si)
	i := new(indexer.Index)
	ic := new(indexer.IdxCollection)
	id := new(indexer.IdxDoc)
	gob.Register(i)
	gob.Register(ic)
	gob.Register(id)
	ss := make([]string, 0)
	gob.Register(ss)
	ms := make(map[string]string)
	gob.Register(ms)
	smsi := make([]map[string]interface{},0)
	gob.Register(smsi)
	msss := make(map[string][]string)
	gob.Register(msss)
	cc := new(client.Client)
	gob.Register(cc)
	uu := new(user.User)
	gob.Register(uu)
	li := new(log_info.LogInfo)
	gob.Register(li)
	mis := map[int]interface{}{}
	gob.Register(mis)
	cbv := new(cookbook.CookbookVersion)
	gob.Register(cbv)
	dbi := new(data_bag.DataBagItem)
	gob.Register(dbi)
}

func setSaveTicker() {
	if config.Config.FreezeData {
		ds := data_store.New()
		ticker := time.NewTicker(time.Second * time.Duration(config.Config.FreezeInterval))
		go func(){
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
				les := log_info.GetLogInfos(0, 1)
				if len(les) != 0 {
					p, err := log_info.PurgeLogInfos(les[0].Id - config.Config.LogEventKeep)
					if err != nil {
						logger.Errorf(err.Error())
					}
					logger.Debugf("Purged %d events automatically", p)
				}
			}
		}()
	}
}
