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

package indexer

import (

)

type PostgresIndex struct {

}

func (p *PostgresIndex) Initialize() error {

	return nil
}

func (p *PostgresIndex) CreateCollection(col string) error {

	return nil
}

func (p *PostgresIndex) DeleteCollection(col string) error {

	return nil
}

func (p *PostgresIndex) DeleteItem(idxName string, doc string) error {

	return nil
}

func (p *PostgresIndex) SaveItem(obj Indexable) error {

	return nil
}

func (p *PostgresIndex) Endpoints() []string {

	return nil
}

func (p *PostgresIndex) Clear() error {

	return nil
}
