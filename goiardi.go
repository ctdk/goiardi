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
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/acl"
	"github.com/ctdk/goiardi/actor"
	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/authentication"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/container"
	"github.com/ctdk/goiardi/cookbook"
	"github.com/ctdk/goiardi/databag"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/environment"
	"github.com/ctdk/goiardi/filestore"
	"github.com/ctdk/goiardi/group"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/loginfo"
	"github.com/ctdk/goiardi/node"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/report"
	"github.com/ctdk/goiardi/role"
	"github.com/ctdk/goiardi/sandbox"
	"github.com/ctdk/goiardi/serfin"
	"github.com/ctdk/goiardi/shovey"
	"github.com/ctdk/goiardi/universe"
	"github.com/ctdk/goiardi/user"
	"github.com/ctdk/goiardi/util"
	"github.com/gorilla/mux"
	serfclient "github.com/hashicorp/serf/client"
	"net/http"
	"os"
	"os/signal"
	"path"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type interceptHandler struct {
	router *mux.Router
}

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
		errch := make(chan error)
		go startEventMonitor(serfin.Serfer, errch)
		err := <-errch
		if err != nil {
			logger.Criticalf(err.Error())
			os.Exit(1)
		}
		startNodeMonitor()
	}

	/* Create default clients and users. Currently chef-validator,
	 * chef-webui, and admin. */
	createDefaultActors()
	handleSignals()

	muxer := mux.NewRouter()
	muxer.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	// may need to set mux.StrictSlash(true)

	/* Register the various handlers, found in their own source files. */
	muxer.HandleFunc("/organizations", orgHandler)
	muxer.HandleFunc("/organizations/{org}", orgMainHandler)
	// This does not seem to be under organizations at all, so far, but
	// on the other hand chef-zero seems to provide for it being there.
	muxer.HandleFunc("/authenticate_user", authenticateUserHandler)
	muxer.HandleFunc("/users", userListHandler)
	muxer.HandleFunc("/users/{name}", userHandler)
	muxer.HandleFunc("/users/{name}/association_requests", userAssocHandler)
	muxer.HandleFunc("/users/{name}/association_requests/count", userAssocCountHandler)
	muxer.HandleFunc("/users/{name}/association_requests/{id}", userAssocIDHandler)
	muxer.HandleFunc("/users/{name}/organizations", userListOrgHandler)
	muxer.HandleFunc("/system_recovery", systemRecoveryHandler)
	// organization routes
	s := muxer.PathPrefix("/organizations/{org}/").Subrouter()
	// get the org tool routes out of the way, out of order
	s.HandleFunc("/association_requests/{id}", orgToolHandler)
	s.HandleFunc("/association_requests", orgToolHandler)
	s.HandleFunc("/_validator_key", orgToolHandler)
	s.HandleFunc("/clients", clientListHandler).Methods("GET")
	s.HandleFunc("/clients", clientCreateHandler).Methods("POST")
	s.HandleFunc("/clients", clientNoMethodHandler)
	s.HandleFunc("/clients/{name}", clientHandler)
	s.HandleFunc("/clients/{name}/_acl", clientACLHandler)
	s.HandleFunc("/clients/{name}/_acl/{perm}", clientACLPermHandler)
	// may be broken up more later
	s.HandleFunc("/containers", containerListHandler)
	s.HandleFunc("/containers/{name}", containerHandler)
	s.HandleFunc("/containers/{name}/_acl", containerACLHandler)
	s.HandleFunc("/containers/{name}/_acl/{perm}", containerACLPermHandler)
	s.HandleFunc("/cookbooks", cookbookHandler)
	s.HandleFunc("/cookbooks/{name}", cookbookHandler)
	s.HandleFunc("/cookbooks/{name}/_acl", cookbookACLHandler)
	s.HandleFunc("/cookbooks/{name}/_acl/{perm}", cookbookACLPermHandler)
	s.HandleFunc("/cookbooks/{name}/{version}", cookbookHandler)
	s.HandleFunc("/data", dataHandler)
	s.HandleFunc("/data/{name}", dataHandler)
	s.HandleFunc("/data/{name}/_acl", dataACLHandler)
	s.HandleFunc("/data/{name}/_acl/{perm}", dataACLPermHandler)
	s.HandleFunc("/data/{name}/{item}", dataHandler)
	s.HandleFunc("/environments", environmentHandler)
	s.HandleFunc("/environments/{name}", environmentHandler)
	s.HandleFunc("/environments/{name}/_acl", environmentACLHandler)
	s.HandleFunc("/environments/{name}/_acl/{perm}", environmentACLPermHandler)
	es := s.PathPrefix("/environments/{name}/").Subrouter()
	es.HandleFunc("/cookbooks", environmentHandler)
	es.HandleFunc("/cookbooks/{op_name}", environmentHandler)
	es.HandleFunc("/cookbook_versions", environmentHandler)
	es.HandleFunc("/nodes", environmentHandler)
	es.HandleFunc("/recipes", environmentHandler)
	es.HandleFunc("/roles/{op_name}", environmentHandler)
	s.HandleFunc("/events", eventListHandler)
	s.HandleFunc("/events/{id}", eventHandler)
	s.HandleFunc("/file_store/{chksum}", fileStoreHandler)
	s.HandleFunc("/groups", groupListHandler)
	s.HandleFunc("/groups/{group_name}", groupHandler)
	s.HandleFunc("/groups/{group_name}/_acl", groupACLHandler)
	s.HandleFunc("/groups/{group_name}/_acl/{perm}", groupACLPermHandler)
	s.HandleFunc("/nodes", nodeListHandler)
	s.HandleFunc("/nodes/{name}", nodeHandler)
	s.HandleFunc("/nodes/{name}/_acl", nodeACLHandler)
	s.HandleFunc("/nodes/{name}/_acl/{perm}", nodeACLPermHandler)
	s.HandleFunc("/organizations/_acl", orgACLHandler)
	s.HandleFunc("/organizations/_acl/{perm}", orgACLEditHandler)
	s.HandleFunc("/principals/{name}", principalHandler)
	s.HandleFunc("/reports/", reportHandler)
	s.HandleFunc("/reports/{foo}", reportHandler)
	s.HandleFunc("/reports/nodes/{node_name}/runs", reportHandler)
	s.HandleFunc("/reports/nodes/{node_name}/runs/{run_id}", reportHandler)
	s.HandleFunc("/reports/org/runs", reportHandler)
	s.HandleFunc("/reports/org/runs/{run_id}", reportHandler)
	s.HandleFunc("/roles", roleListHandler)
	s.HandleFunc("/roles/{name}", roleHandler)
	s.HandleFunc("/roles/{name}/_acl", roleACLHandler)
	s.HandleFunc("/roles/{name}/_acl/{perm}", roleACLPermHandler)
	s.HandleFunc("/roles/{name}/environments", roleHandler)
	s.HandleFunc("/roles/{name}/environments/{env_name}", roleHandler)
	s.HandleFunc("/sandboxes", sandboxHandler)
	s.HandleFunc("/sandboxes/{id}", sandboxHandler)
	s.Path("/search/reindex").HandlerFunc(reindexHandler)
	s.HandleFunc("/search", searchHandler)
	s.HandleFunc("/search/{index}", searchHandler)
	s.HandleFunc("/shovey/jobs", shoveyHandler)
	s.HandleFunc("/shovey/jobs/{job_id}", shoveyHandler)
	s.HandleFunc("/shovey/jobs/{job_id}/{node_name}", shoveyHandler)
	s.HandleFunc("/shovey/stream/{job_id}/{node_name}", shoveyHandler)
	s.HandleFunc("/status/{specif}/nodes", statusHandler)
	s.HandleFunc("/status/{specif}/{node_name}/{op}", statusHandler)
	s.HandleFunc("/users", userOrgListHandler)
	s.HandleFunc("/users/{name}", userOrgHandler)
	s.HandleFunc("/universe", universe.UniverseHandler)

	/* TODO: figure out how to handle the root & not found pages */
	muxer.HandleFunc("/", rootHandler)

	h := &interceptHandler{router: muxer}

	listenAddr := config.ListenAddr()
	var err error
	srv := &http.Server{Addr: listenAddr, Handler: h}
	if config.Config.UseSSL {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS10}
		err = srv.ListenAndServeTLS(config.Config.SSLCert, config.Config.SSLKey)
	} else {
		err = srv.ListenAndServe()
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

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jsonErrorReport(w, r, "not found 12345", http.StatusNotFound)
	return
}

func (h *interceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/* knife sometimes sends URL paths that start with //. Redirecting
	 * worked for GETs, but since it was breaking POSTs and screwing with
	 * GETs with query params, we just clean up the path and move on. */

	/* log the URL */
	logger.Debugf("Serving %s -- %s", r.URL.Path, r.Method)

	if r.Method != "CONNECT" {
		if p := cleanPath(r.URL.Path); p != r.URL.Path {
			r.URL.Path = p
		}
	}

	fstorere := regexp.MustCompile(`^/organizations/[^/]*/file_store`)
	/* Make configurable, I guess, but Chef wants it to be 1000000 */
	if !fstorere.MatchString(r.URL.Path) && r.ContentLength > config.Config.JSONReqMaxSize {
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
		pathArray := strings.Split(r.URL.Path[1:], "/")
		var org *organization.Organization
		if pathArray[0] == "organizations" {
			var err util.Gerror
			org, err = organization.Get(pathArray[1])
			if err != nil {
				jsonErrorReport(w, r, err.Error(), err.Status())
				return
			}
		}
		if _, uherr := actor.GetReqUser(org, userID); uherr != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Warningf("Attempting to use invalid user %s through X-Ops-Request-Source = web", userID)
			jsonErrorReport(w, r, "invalid action", http.StatusUnauthorized)
			return
		}
		userID = "pivotal"
	}
	/* Only perform the authorization check if that's configured. Bomb with
	 * an error if the check of the headers, timestamps, etc. fails. */
	/* No clue why /principals doesn't require authorization. Hrmph. */
	princre := regexp.MustCompile(`/organizations/[^/]*/principals`)
	if config.Config.UseAuth && !fstorere.MatchString(r.URL.Path) && !(princre.MatchString(r.URL.Path) && r.Method == "GET") {
		herr := authentication.CheckHeader(userID, r)
		if herr != nil {
			w.Header().Set("Content-Type", "application/json")
			logger.Errorf("Authorization failure: %s\n", herr.Error())
			w.Header().Set("Www-Authenticate", `X-Ops-Sign version="1.0" version="1.1" version="1.2"`)
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

	//http.DefaultServeMux.ServeHTTP(w, r)
	// Now instead of using the default ServeHTTP, we use the gorilla mux
	// one. We aren't able to use it directly, however, because the chef
	// clients and knife get unhappy unless we're able to do the above work
	// before serving the reuquests.
	h.router.ServeHTTP(w, r)
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

// TODO: this has to change for organizations.
func createDefaultActors() {
	cworg, _ := organization.Get("default")
	if cworg == nil {
		if org, oerr := organization.New("default", "default org"); oerr != nil {
			logger.Criticalf(oerr.Error())
			os.Exit(1)
		} else {
			err := org.Save()
			if err != nil {
				logger.Criticalf(err.Error())
				os.Exit(1)
			}
			cworg = org
		}
	}
	if cwebui, _ := client.Get(cworg, "default-webui"); cwebui == nil {
		if webui, nerr := client.New(cworg, "default-webui"); nerr != nil {
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

	if cvalid, _ := client.Get(cworg, "default-validator"); cvalid == nil {
		if validator, verr := client.New(cworg, "default-validator"); verr != nil {
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

	// the admin user is called 'pivotal' now with chef12 for some reason.
	if uadmin, _ := user.Get("pivotal"); uadmin == nil {
		if admin, aerr := user.New("pivotal"); aerr != nil {
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

	environment.MakeDefaultEnvironment(cworg)

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
	o := new(organization.Organization)
	gob.Register(o)
	gob.Register(new(association.AssociationReq))
	gob.Register(new(association.Association))
	gob.Register(new(group.Group))
	gob.Register(new(container.Container))
	gob.Register(new(acl.ACL))
	gob.Register(new(acl.ACLitem))
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
				orgs := organization.AllOrganizations()
				var purged int64
				for _, org := range orgs {
					les, _ := loginfo.GetLogInfos(org, nil, 0, 1)
					if len(les) != 0 {
						p, err := loginfo.PurgeLogInfos(org, les[0].ID-config.Config.LogEventKeep)
						if err != nil {
							logger.Errorf(err.Error())
						}
						purged += p
					}
				}
				if purged != 0 {
					logger.Debugf("Purged %d events automatically", purged)
				}
			}
		}()
	}
}

func startEventMonitor(sc *serfclient.RPCClient, errch chan<- error) {
	ch := make(chan map[string]interface{}, 10)
	sh, err := sc.Stream("*", ch)
	if err != nil {
		errch <- err
		return
	}
	errch <- nil

	defer sc.Stop(sh)
	// watch the events and queries
	for e := range ch {
		logger.Debugf("Got an event: %v", e)
		eName, _ := e["Name"]
		switch eName {
		case "node_status":
			jsonPayload := make(map[string]string)
			err = json.Unmarshal(e["Payload"].([]byte), &jsonPayload)
			if err != nil {
				logger.Errorf(err.Error())
				continue
			}
			org, err := organization.Get(jsonPayload["organization"])
			if err != nil {
				logger.Errorf(err.Error())
				continue
			}
			n, _ := node.Get(org, jsonPayload["node"])
			if n == nil {
				logger.Errorf("No node %s", jsonPayload["node"])
				continue
			}
			nerr := n.UpdateStatus(jsonPayload["status"])
			if nerr != nil {
				logger.Errorf(nerr.Error())
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
	}
	return
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
