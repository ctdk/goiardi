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

package search

import (
	"github.com/ctdk/goas/v2/logger"
	"github.com/ctdk/goiardi/indexer"
	//"github.com/ctdk/goiardi/util"
)

type PostgresSearch struct {
}

func (p *PostgresSearch) Search(idx string, q string, rows int, sortOrder string, start int, partialData map[string]interface{}) ([]map[string]interface{}, error) {
	// keep up with the ersatz solr.
	qq := &Tokenizer{Buffer: q}
	qq.Init()
	if err := qq.Parse(); err != nil {
		return nil, err
	}
	qq.Execute()
	qchain := qq.Evaluate()

	logger.Debugf("what on earth is the chain? %q", qchain)

	// dummy
	dres := make([]map[string]interface{}, 0)
	return dres, nil
}

func (p *PostgresSearch) GetEndpoints() []string {
	// TODO: deal with possible errors
	endpoints, err := indexer.Endpoints()
	return endpoints
	if err != nil {
		panic(err)
	}
	return endpoints
}
