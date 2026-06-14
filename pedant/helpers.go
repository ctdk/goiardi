// Package pedant provides integration test helpers for goiardi, ported from
// chef-pedant. These tests start an in-memory goiardi server and exercise
// the Chef Server API against it.
package pedant

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestServer wraps an httptest.Server with pre-configured requestors.
type TestServer struct {
	Server          *http.Server
	BaseURL         string
	AdminUser       *TestRequestor
	NormalUser      *TestRequestor
	AdminClient     *TestRequestor
	NormalClient    *TestRequestor
	ValidatorClient *TestRequestor
	Superuser       *TestRequestor
}

// TestRequestor holds a name and private key for signing Chef requests.
type TestRequestor struct {
	Name        string
	PrivateKey  *rsa.PrivateKey
	IsUser      bool
	IsAdmin     bool
	IsValidator bool
}

// ChefSigningClient makes signed requests to the goiardi test server.
type ChefSigningClient struct {
	Requestor  *TestRequestor
	BaseURL    string
	HTTPClient *http.Client
}

// Response wraps an HTTP response for test assertions.
type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

// NewClient creates a ChefSigningClient for the given requestor.
func (ts *TestServer) NewClient(r *TestRequestor) *ChefSigningClient {
	return &ChefSigningClient{
		Requestor: r,
		BaseURL:   ts.BaseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIURL constructs a full API URL from a path fragment.
func (ts *TestServer) APIURL(pathFragment string) string {
	if !strings.HasPrefix(pathFragment, "/") {
		pathFragment = "/" + pathFragment
	}
	return ts.BaseURL + pathFragment
}

// Get performs a signed GET request.
func (c *ChefSigningClient) Get(path string) (*Response, error) {
	return c.doRequest("GET", path, nil)
}

// Post performs a signed POST request with the given body.
func (c *ChefSigningClient) Post(path string, body interface{}) (*Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
	}
	return c.doRequest("POST", path, bodyBytes)
}

// Put performs a signed PUT request with the given body.
func (c *ChefSigningClient) Put(path string, body interface{}) (*Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
	}
	return c.doRequest("PUT", path, bodyBytes)
}

// Delete performs a signed DELETE request.
func (c *ChefSigningClient) Delete(path string) (*Response, error) {
	return c.doRequest("DELETE", path, nil)
}

func (c *ChefSigningClient) doRequest(method, path string, body []byte) (*Response, error) {
	u := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Sign the request
	c.signRequest(req, body)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Header:     resp.Header,
	}, nil
}

func (c *ChefSigningClient) signRequest(req *http.Request, body []byte) {
	// Calculate content hash
	var bodyStr string
	if len(body) > 0 {
		bodyStr = string(body)
	}
	contentHash := hashStr(bodyStr)

	// Set standard headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Chef-Version", "11.12.0")
	req.Header.Set("X-Ops-Content-Hash", contentHash)
	req.Header.Set("X-Ops-Timestamp", time.Now().UTC().Format(time.RFC3339))
	req.Header.Set("X-Ops-UserId", c.Requestor.Name)
	req.Header.Set("X-Ops-Sign", "algorithm=sha1;version=1.0")

	// Build the string to sign
	hashedPath := hashStr(req.URL.Path)
	content := fmt.Sprintf("Method:%s\nHashed Path:%s\nX-Ops-Content-Hash:%s\nX-Ops-Timestamp:%s\nX-Ops-UserId:%s",
		req.Method, hashedPath, contentHash,
		req.Header.Get("X-Ops-Timestamp"),
		req.Header.Get("X-Ops-UserId"),
	)

	// Generate signature
	signature, err := privateEncrypt(c.Requestor.PrivateKey, []byte(content))
	if err != nil {
		panic(fmt.Sprintf("signing request: %v", err))
	}

	// Base64 encode and split into 60-char chunks
	base64sig := base64.StdEncoding.EncodeToString(signature)
	for i := 0; i*60 < len(base64sig); i++ {
		end := (i + 1) * 60
		if end > len(base64sig) {
			end = len(base64sig)
		}
		req.Header.Set(fmt.Sprintf("X-Ops-Authorization-%d", i+1), base64sig[i*60:end])
	}
}

// SignRawRequest signs an already-constructed HTTP request with Chef auth headers.
// This is useful for sending raw payloads that can't be JSON-marshaled.
func (c *ChefSigningClient) SignRawRequest(req *http.Request, body []byte) {
	c.signRequest(req, body)
}

func hashStr(toHash string) string {
	h := sha1.New()
	io.WriteString(h, toHash)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func privateEncrypt(key *rsa.PrivateKey, data []byte) ([]byte, error) {
	k := (key.N.BitLen() + 7) / 8
	tLen := len(data)

	if tLen > k-11 {
		return nil, fmt.Errorf("data too long")
	}
	em := make([]byte, k)
	em[1] = 1
	for i := 2; i < k-tLen-1; i++ {
		em[i] = 0xff
	}
	copy(em[k-tLen:k], data)
	c := new(big.Int).SetBytes(em)
	if c.Cmp(key.N) > 0 {
		return nil, nil
	}
	var m *big.Int
	var m2 *big.Int
	if key.Precomputed.Dp == nil {
		m = new(big.Int).Exp(c, key.D, key.N)
	} else {
		m = new(big.Int).Exp(c, key.Precomputed.Dp, key.Primes[0])
		m2 = new(big.Int).Exp(c, key.Precomputed.Dq, key.Primes[1])
		m.Sub(m, m2)
		if m.Sign() < 0 {
			m.Add(m, key.Primes[0])
		}
		m.Mul(m, key.Precomputed.Qinv)
		m.Mod(m, key.Primes[0])
		m.Mul(m, key.Primes[1])
		m.Add(m, m2)

		for i, values := range key.Precomputed.CRTValues {
			prime := key.Primes[2+i]
			m2.Exp(c, values.Exp, prime)
			m2.Sub(m2, m)
			m2.Mul(m2, values.Coeff)
			m2.Mod(m2, prime)
			if m2.Sign() < 0 {
				m2.Add(m2, prime)
			}
			m2.Mul(m2, values.R)
			m.Add(m, m2)
		}
	}
	return m.Bytes(), nil
}

// --- Test assertion helpers ---

// AssertStatus checks that the response has the expected status code.
func AssertStatus(t *testing.T, resp *Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(resp.Body))
	}
}

// AssertBodyContains checks that the response body contains the expected string.
func AssertBodyContains(t *testing.T, resp *Response, substr string) {
	t.Helper()
	if !bytes.Contains(resp.Body, []byte(substr)) {
		t.Errorf("expected body to contain %q, got: %s", substr, string(resp.Body))
	}
}

// AssertBodyNotContains checks that the response body does not contain the given string.
func AssertBodyNotContains(t *testing.T, resp *Response, substr string) {
	t.Helper()
	if bytes.Contains(resp.Body, []byte(substr)) {
		t.Errorf("expected body to NOT contain %q, got: %s", substr, string(resp.Body))
	}
}

// ParseJSON parses the response body as JSON into the given target.
func ParseJSON(t *testing.T, resp *Response, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(resp.Body, target); err != nil {
		t.Fatalf("parsing JSON response: %v\nBody: %s", err, string(resp.Body))
	}
}

// GetJSONBody parses the response body into a map.
func GetJSONBody(t *testing.T, resp *Response) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	ParseJSON(t, resp, &body)
	return body
}

// GetJSONArray parses the response body into a slice.
func GetJSONArray(t *testing.T, resp *Response) []interface{} {
	t.Helper()
	var body []interface{}
	ParseJSON(t, resp, &body)
	return body
}

// --- Payload builders ---

// NewNode creates a node payload.
func NewNode(name string, opts ...map[string]interface{}) map[string]interface{} {
	n := map[string]interface{}{
		"name":             name,
		"json_class":       "Chef::Node",
		"chef_type":        "node",
		"chef_environment": "_default",
		"override":         map[string]interface{}{},
		"normal":           map[string]interface{}{},
		"default":          map[string]interface{}{},
		"automatic":        map[string]interface{}{},
		"run_list":         []string{},
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			n[k] = v
		}
	}
	return n
}

// NewRole creates a role payload.
func NewRole(name string, opts ...map[string]interface{}) map[string]interface{} {
	r := map[string]interface{}{
		"name":                name,
		"json_class":          "Chef::Role",
		"chef_type":           "role",
		"default_attributes":  map[string]interface{}{},
		"override_attributes": map[string]interface{}{},
		"run_list":            []string{},
		"env_run_lists":       map[string]interface{}{},
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			r[k] = v
		}
	}
	return r
}

// NewEnvironment creates an environment payload.
func NewEnvironment(name string, opts ...map[string]interface{}) map[string]interface{} {
	e := map[string]interface{}{
		"name":                name,
		"json_class":          "Chef::Environment",
		"chef_type":           "environment",
		"description":         "",
		"default_attributes":  map[string]interface{}{},
		"override_attributes": map[string]interface{}{},
		"cookbook_versions":   map[string]string{},
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			e[k] = v
		}
	}
	return e
}

// NewDataBag creates a data bag payload.
func NewDataBag(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":       name,
		"json_class": "Chef::DataBag",
		"chef_type":  "data_bag",
	}
}

// NewDataBagItem creates a data bag item payload.
func NewDataBagItem(id string, data ...map[string]interface{}) map[string]interface{} {
	item := map[string]interface{}{
		"id":         id,
		"json_class": "Chef::DataBagItem",
		"chef_type":  "data_bag_item",
	}
	if len(data) > 0 {
		for k, v := range data[0] {
			item[k] = v
		}
	}
	return item
}

// NewClient creates a client payload.
func NewClient(name string, opts ...map[string]interface{}) map[string]interface{} {
	c := map[string]interface{}{
		"name":      name,
		"admin":     false,
		"validator": false,
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			c[k] = v
		}
	}
	return c
}

// NewUser creates a user payload.
func NewUser(name string, opts ...map[string]interface{}) map[string]interface{} {
	u := map[string]interface{}{
		"name":     name,
		"password": "foobar",
		"admin":    false,
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			u[k] = v
		}
	}
	return u
}

// UniqueName generates a unique name with the given prefix.
func UniqueName(prefix string) string {
	ts := time.Now().UnixNano()
	return fmt.Sprintf("pedant_%s_%d-%d", prefix, ts, os.Getpid())
}

// --- Additional helpers for test assertions ---

// AssertResponseMatches checks that the response matches expected status and body.
func AssertResponseMatches(t *testing.T, resp *Response, expectedStatus int, expectedBody map[string]interface{}) {
	t.Helper()
	AssertStatus(t, resp, expectedStatus)
	if expectedBody != nil {
		body := GetJSONBody(t, resp)
		for k, v := range expectedBody {
			got, ok := body[k]
			if !ok {
				t.Errorf("expected key %q in response body, not found", k)
				continue
			}
			// Compare as JSON
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(v)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("key %q: expected %s, got %s", k, string(wantJSON), string(gotJSON))
			}
		}
	}
}

// AssertErrorResponse checks that the response is an error with the expected status and message.
func AssertErrorResponse(t *testing.T, resp *Response, expectedStatus int, expectedError string) {
	t.Helper()
	AssertStatus(t, resp, expectedStatus)
	body := GetJSONBody(t, resp)
	errs, ok := body["error"]
	if !ok {
		t.Errorf("expected error response with key 'error', got: %s", string(resp.Body))
		return
	}
	errStr := fmt.Sprintf("%v", errs)
	if !strings.Contains(errStr, expectedError) {
		t.Errorf("expected error to contain %q, got %q", expectedError, errStr)
	}
}

// AssertURIMatches checks that the response contains a "uri" field matching the expected path.
func AssertURIMatches(t *testing.T, ts *TestServer, resp *Response, expectedPath string) {
	t.Helper()
	body := GetJSONBody(t, resp)
	uri, ok := body["uri"]
	if !ok {
		t.Errorf("expected 'uri' in response body, got: %s", string(resp.Body))
		return
	}
	expectedURI := ts.APIURL(expectedPath)
	if uri != expectedURI {
		t.Errorf("expected uri %q, got %q", expectedURI, uri)
	}
}

// AssertBodyExact checks that the response body exactly matches the expected JSON.
func AssertBodyExact(t *testing.T, resp *Response, expected interface{}) {
	t.Helper()
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("marshaling expected: %v", err)
	}
	// Normalize both by unmarshaling and remarshaling
	var got, want interface{}
	if err := json.Unmarshal(resp.Body, &got); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if err := json.Unmarshal(expectedJSON, &want); err != nil {
		t.Fatalf("unmarshaling expected: %v", err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("body mismatch:\ngot:  %s\nwant: %s", string(gotJSON), string(wantJSON))
	}
}

// NormalizeRunList normalizes a run list (wraps bare recipe names in recipe[]).
func NormalizeRunList(runList []string) []string {
	result := make([]string, 0, len(runList))
	seen := make(map[string]bool)
	for _, item := range runList {
		var normalized string
		if strings.HasPrefix(item, "recipe[") && strings.HasSuffix(item, "]") {
			normalized = item
		} else if strings.HasPrefix(item, "role[") && strings.HasSuffix(item, "]") {
			normalized = item
		} else {
			normalized = "recipe[" + item + "]"
		}
		if !seen[normalized] {
			result = append(result, normalized)
			seen[normalized] = true
		}
	}
	return result
}
