/*
 * Copyright (c) 2013-2026, Jeremy Bingham (<jeremy@goiardi.gl>)
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

// Package infinity defines a couple of variables for "infinite" timestamps. Put
// here for easier sharing where needed.
package infinity

import (
	"math"
	"time"
)

// It's not quite infinity, but it's close enough. Client and user keys can have
// "infinity" as an expiration date, which are supported as special values by
// the Postgres DB driver, but since golang doesn't provide a convenient way to
// handle that directly we'll do the next best thing: use dates ~292 billion
// years in the past and future to represent "infinity" and "-infinity". The
// Year 292,277,026,596 Problem will need to be dealt with sometime in the next
// several aeons.

// Infinity is the next best thing to an infinite date in the future we can
// currently have.
var Infinity = time.Unix(math.MaxInt64, 0)

// MinusInfinity is, similarly, the next best thing to a date in the infinite
// past that we can have. Since the universe itself is not nearly as old as this
// date, it should not be a problem given our current understanding of the
// history of the universe and physics.
var MinusInfinity = time.Unix(math.MinInt64, 0)
