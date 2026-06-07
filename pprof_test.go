/*
 * Copyright (c) 2013-2017, Jeremy Bingham (<jeremy@goiardi.gl>)
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

import "testing"

func TestPprofAllowed(t *testing.T) {
	// The whitelist is empty in this test, so only loopback peers are
	// trusted. These cases verify that an attacker-supplied X-Forwarded-For
	// header cannot bypass the loopback gate.
	cases := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		want          bool
	}{
		{"loopback peer, no XFF", "127.0.0.1:5000", "", true},
		{"ipv6 loopback peer, no XFF", "[::1]:5000", "", true},
		{"remote peer, no XFF", "203.0.113.7:5000", "", false},
		// The core vulnerability: a remote peer spoofs a loopback
		// X-Forwarded-For value. It must still be blocked.
		{"remote peer spoofs loopback XFF", "203.0.113.7:5000", "127.0.0.1", false},
		{"remote peer spoofs loopback XFF in chain", "203.0.113.7:5000", "10.0.0.1, 127.0.0.1", false},
		// A trusted (loopback) reverse proxy forwarding a real client:
		// the forwarded client address governs the decision.
		{"loopback proxy, remote client", "127.0.0.1:5000", "203.0.113.7", false},
		{"loopback proxy, loopback client", "127.0.0.1:5000", "127.0.0.1", true},
		// Malformed input must fail closed.
		{"bad remote addr", "not-an-address", "", false},
		{"loopback proxy, garbage XFF", "127.0.0.1:5000", "garbage", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pprofAllowed(tc.remoteAddr, tc.xForwardedFor); got != tc.want {
				t.Errorf("pprofAllowed(%q, %q) = %v, want %v", tc.remoteAddr, tc.xForwardedFor, got, tc.want)
			}
		})
	}
}
