/* Search functions */

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
)

func search_handler(w http.ResponseWriter, r *http.Request){
	/* Search is easy, at least for the moment. Totally not implemented, so
	 * send an error saying that back. Someday this will work. */
	JsonErrorReport(w, r, "Sorry, at the moment search is totally not implemented. Someday, perhaps, but that day is not today.", http.StatusNotImplemented)
}
