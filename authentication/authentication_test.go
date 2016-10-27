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

package authentication

import (
	"net/url"
	"testing"
	"time"

	"github.com/go-chef/chef"
)

var privKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEA1uhuCvIwWIqDljof4IlGz4k9/pwirF7YSLiZ5U4UCTaNcpCG
Bv1O3Lk1BXxJbE3ADGQ9P3Fia5I8oH6KzIY/Y9DXxVa18j+PJaDsgbLI8ppQz3Wx
RCpc/WRxKCfJklyB48dxP3NdW4Yl1AzI4jKkkBxRWmG70awxtNg9TZbLCeIcFW1T
pbIAY2sdRgxWrkYSAO5zjN++TGN2wONfMiqZLlCay/U2KqTQha9urjWncBu+/Vzl
f/21jNB6XUvKlaJ7Pdne5A7W7uUviKPER7SIPFmMSI5pE2kDaSEbuU1NOSa6sjU0
tVywpSuXNWvrgjdbpvOXPqs8CMtDQYxSDGD05wIDAQABAoIBADdh1KH7gdv/biOz
vO1HUzk+e/x2TjUvh/tNn1NJiL5LEa6ZcgCxHLai//f27JD9hGVtG5+S37MrD3ao
xaopWoKlmkVfsCnKmWAwFWLjKQfkHrkn8lPHuwkN7l9TyY1vS4Xgqt2YJFHmwy7f
UJGCTYhZ09k/4IALKRAavcmV96MG6bsLlZ+p5OCz96HqYx+knrc2fQaWY9fSjHUY
nmgj6YZ4E4Hw4+E3t6WLbYDb52wVwXBAFzWC7fmSPzxLzFgIM4LHXFh9t8bHQuVd
/76H9wcd5o4+kOiquOWieMvJwPqtE5VXUwKycM9OxSZJ6SSLDfBQxQKW0WaLkf9t
oN0GmGECgYEA/p1z1/xd+UZbGaeZyczvqfo7YOT16MNi9c/oba1yS+9h7vM9KJeP
KfJT4VAh5fMg9j4wpn4KtA7H6vCsYaVwhFwNLv4GkMIMIHAdY2G8kCphB9hhMlM0
Pc/+gxnXvcPwFRswc+qtbLCpXTBxrT8p68gctdL4GLOak3LbzrxWGhECgYEA2BOv
kEHSrZkCnAwoxtfaRhNDIQJainxW2nf5c0m4kIKl8cytecEMkr7ImL8kOmMOnic8
/Q/kbouJIbBVAXlGJXZVLmCgqqIvDdfkx5G1+oooc4UXr/6WB+h7CQ85dpeyVJxT
P/7hmMgtsyjeyTD/Hug2GM1F6T4a9Rf7rQ/+Z3cCgYAmRo0vnuSRoJ35UVSxHXm5
18AtZL4C67xor4SFWFmiSK40OaSsAXyoFaG+cUlnRBFkcxzlKnV5c+9hxiRj2Xb8
rsnckptyD3m7Np90XTD3iydjAog6BIAJ+saL9sqT4Gyq/5ddFZ5UhIoxVCMCpEgt
BbwrKTfansVR/SZGAdH/wQKBgEPrLCw0BHz8s41JZSfbgYi1VUxy6PLO0p4pSAet
DI6gAnlW1NCIleMqhPM+YazYpiegPdNtw2fcBGbKfm3QKPRtlajWRqpcAF5hllAE
xSbTdpOZKjDv3UjvEn1ug6l7VVqzKJfdDhxwD61ZE246MHcOlrKFE4yVMPQJbdqg
RF9RAoGATQWBtuimFHqdHXqEBgTIuPiSgdRUOsSeAieljl3KOebvRZhcIA5yrwzH
TcLn7ZwkLlfDzQzeqsn57lhrG9VaI28d4cQLJyDFQowvmfP39XInO4gEvSGM6n+K
/FTYRAAOCaQOs55oL0l0Opmk/EL0ltf/zmnvcQRDfrQKc3HyzYg=
-----END RSA PRIVATE KEY-----
`

var pubKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1uhuCvIwWIqDljof4IlG
z4k9/pwirF7YSLiZ5U4UCTaNcpCGBv1O3Lk1BXxJbE3ADGQ9P3Fia5I8oH6KzIY/
Y9DXxVa18j+PJaDsgbLI8ppQz3WxRCpc/WRxKCfJklyB48dxP3NdW4Yl1AzI4jKk
kBxRWmG70awxtNg9TZbLCeIcFW1TpbIAY2sdRgxWrkYSAO5zjN++TGN2wONfMiqZ
LlCay/U2KqTQha9urjWncBu+/Vzlf/21jNB6XUvKlaJ7Pdne5A7W7uUviKPER7SI
PFmMSI5pE2kDaSEbuU1NOSa6sjU0tVywpSuXNWvrgjdbpvOXPqs8CMtDQYxSDGD0
5wIDAQAB
-----END PUBLIC KEY-----
`

func TestTimeSlewSuccess(t *testing.T) {
	dur, _ := time.ParseDuration("15m")
	nowTime := time.Now().UTC()
	nowStr := nowTime.Format(time.RFC3339)
	tok, terr := checkTimeStamp(nowStr, dur)
	if !tok {
		t.Errorf("Time %s should have been OK, but it wasn't!", terr.Error())
	}
	fiveMin := time.Duration(5) * time.Minute
	fiveMinTime := nowTime.Add(fiveMin)
	fStr := fiveMinTime.Format(time.RFC3339)
	tok, terr = checkTimeStamp(fStr, dur)
	if !tok {
		t.Errorf("Time %s five minutes in the future should have been OK, but it wasn't!", terr)
	}
	fiveMinAgoTime := nowTime.Add(-fiveMin)
	fStr = fiveMinAgoTime.Format(time.RFC3339)
	tok, terr = checkTimeStamp(fStr, dur)
	if !tok {
		t.Errorf("Time %s five minutes in the past should have been OK, but it wasn't!", terr)
	}
}

func TestTimeSlewFail(t *testing.T) {
	dur, _ := time.ParseDuration("15m")
	nowTime := time.Now().UTC()
	oneHour := time.Duration(1) * time.Hour
	oneHourTime := nowTime.Add(oneHour)
	oStr := oneHourTime.Format(time.RFC3339)
	tok, terr := checkTimeStamp(oStr, dur)
	if tok {
		t.Errorf("Time %s one hour in the future should have failed, but didn't", terr)
	}
	oneHourAgoTime := nowTime.Add(-oneHour)
	oStr = oneHourAgoTime.Format(time.RFC3339)
	tok, terr = checkTimeStamp(oStr, dur)
	if tok {
		t.Errorf("Time %s one hour in the past should have failed, but didn't", terr)
	}
}

func TestAuthenticateHeader(t *testing.T) {
	base, _ := url.Parse("https://localhost/")
	pkey, _ := chef.PrivateKeyFromString([]byte(privKey))

	chefClient := &chef.Client{
		BaseURL: base,
		Auth: &chef.AuthConfig{
			PrivateKey: pkey,
			ClientName: "testClient",
		},
	}

	req, err := chefClient.NewRequest("GET", "clients", nil)

	if err != nil {
		t.Errorf("failed to create chef client")
	}

	if err = AuthenticateHeader(pubKey, time.Duration(0), req); err != nil {
		t.Errorf("header authentication failed: %s", err.Error())
	}
}
