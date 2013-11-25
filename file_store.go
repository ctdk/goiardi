/* Local server file storage, when we want to keep files locally and not load
 * them up to S3. Pretty much the same sort of thing chef-zero does here. */

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
	"github.com/ctdk/goiardi/filestore"
	"fmt"
	"encoding/json"
)

func file_store_handler(w http.ResponseWriter, r *http.Request){
	/* We *don't* always set the the content-type to application/json here,
	 * for obvious reasons. Still do for the PUT/POST though. */
	chksum := r.URL.Path[12:]
	
	switch r.Method {
		case "GET":
			w.Header().Set("Content-Type", "application/x-binary")
			file_store, err := filestore.Get(chksum)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Write(*file_store.Data)
		case "PUT", "POST": /* Seems like for file uploads we ought to
				     * support POST too. */
			w.Header().Set("Content-Type", "application/json")
			/* Need to distinguish file already existing and some
			 * sort of error with uploading the file. */
			if file_store, _ := filestore.Get(chksum); file_store != nil {
				file_err := fmt.Errorf("File with checksum %s already exists.", chksum)
				JsonErrorReport(w, r, file_err.Error(), http.StatusConflict)
				return
			}
			file_store, err := filestore.New(chksum, r.Body, r.ContentLength)
			if err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
				return
			}
			file_store.Save()
			file_response := make(map[string]string)
			file_response[file_store.Chksum] = fmt.Sprintf("File with checksum %s uploaded.", file_store.Chksum)
			enc := json.NewEncoder(w)
			if err := enc.Encode(&file_response); err != nil {
				JsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			}
		/* Add DELETE later? */
		default:
			JsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
	}
}
