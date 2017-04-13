/* Local server file storage, when we want to keep files locally and not load
 * them up to S3. Pretty much the same sort of thing chef-zero does here. */

/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"encoding/json"
	"fmt"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/filestore"
	"net/http"
)

func fileStoreHandler(w http.ResponseWriter, r *http.Request) {
	/* We *don't* always set the the content-type to application/json here,
	 * for obvious reasons. Still do for the PUT/POST though. */
	chksum := r.URL.Path[12:]

	/* Eventually, both local storage (in-memory or on disk, depending) or
	 * uploading to s3 or a similar cloud storage provider needs to be
	 * supported. */
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		w.Header().Set("Content-Type", "application/x-binary")
		fileStore, err := filestore.Get(chksum)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if r.Method == http.MethodHead {
			headResponse(w, r, http.StatusOK)
			return
		}
		w.Write(*fileStore.Data)
	case http.MethodPut, http.MethodPost: /* Seems like for file uploads we ought to
		 * support POST too. */
		w.Header().Set("Content-Type", "application/json")
		/* Need to distinguish file already existing and some
		 * sort of error with uploading the file. */
		if fileStore, _ := filestore.Get(chksum); fileStore != nil {
			fileErr := fmt.Errorf("File with checksum %s already exists.", chksum)
			/* Send status OK. It seems chef-pedant at least
			 * tries to upload files twice for some reason.
			 */
			jsonErrorReport(w, r, fileErr.Error(), http.StatusOK)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, config.Config.ObjMaxSize)
		fileStore, err := filestore.New(chksum, r.Body, r.ContentLength)
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		err = fileStore.Save()
		if err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
			return
		}
		fileResponse := make(map[string]string)
		fileResponse[fileStore.Chksum] = fmt.Sprintf("File with checksum %s uploaded.", fileStore.Chksum)
		enc := json.NewEncoder(w)
		if err := enc.Encode(&fileResponse); err != nil {
			jsonErrorReport(w, r, err.Error(), http.StatusInternalServerError)
		}
	/* Add DELETE later? */
	default:
		jsonErrorReport(w, r, "Unrecognized method!", http.StatusMethodNotAllowed)
	}
}
