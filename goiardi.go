/* A relatively simple Chef server implementation in Go, as a learning project
 * to learn more about programming in Go. */

/*
 * Copyright (c) 2013, Jeremy Bingham (<jbingham@gmail.com>)
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
	"log"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/actor"
)

type InterceptHandler struct {} // Doesn't need to do anything, just sit there.

func main(){
	config.ParseConfigOptions()

	/* Create default clients and users. Currently chef-validator,
	 * chef-webui, and admin. */
	createDefaultActors()

	/* Register the various handlers, found in their own source files. */
	http.HandleFunc("/authenticate_user", authenticate_user_handler)
	http.HandleFunc("/clients", list_handler)
	http.HandleFunc("/clients/", actor_handler)
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
	http.HandleFunc("/users", list_handler)
	http.HandleFunc("/users/", actor_handler)
	http.HandleFunc("/file_store/", file_store_handler)

	/* TODO: figure out how to handle the root & not found pages */
	http.HandleFunc("/", root_handler)

	listen_addr := config.ListenAddr()
	http.ListenAndServe(listen_addr, &InterceptHandler{})
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
	//log.Printf("Serving %s\n", r.URL.Path)

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
	if webui, err := actor.New("chef-webui", "client"); err != nil {
		log.Fatalln(err)
	} else {
		webui.Admin = true
		_, err = webui.GenerateKeys()
		if err != nil {
			log.Fatalln(err)
		}
		webui.Save()
	}

	if validator, err := actor.New("chef-validator", "client"); err != nil {
		log.Fatalln(err)
	} else {
		validator.Validator = true
		_, err = validator.GenerateKeys()
		if err != nil {
			log.Fatalln(err)
		}
		validator.Save()
	}

	if admin, err := actor.New("admin", "user"); err != nil {
		log.Fatalln(err)
	} else {
		admin.Admin = true
		_, err = admin.GenerateKeys()
		if err != nil {
			log.Fatalln(err)
		}
		admin.Save()
	}

	return
}
