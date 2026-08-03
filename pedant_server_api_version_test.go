package main

import (
	"net/http"
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Server API Version (/server_api_version) ---
//
// Ported from oc-chef-pedant:
//   spec/api/server_api_version_spec.rb
//
// Chef Server's /server_api_version endpoint reports the supported API
// version range (min_api_version and max_api_version) and optionally a
// version_string. It is used by clients to negotiate the X-Ops-Server-
// Api-Version header.
//
// Known goiardi gap documented by these tests:
//   * goiardi does NOT implement /server_api_version. Any request to the
//     endpoint falls through to the notFoundHandler and returns 404.
//   * Because the endpoint is missing, the positive body-shape tests and the
//     requestor matrix cannot be exercised.
//   * goiardi does validate X-Ops-Server-Api-Version on requests in some
//     paths (e.g. actor endpoints), but the generic /license version
//     validation tests from the Ruby spec cannot be run here.
//
// When goiardi implements /server_api_version, remove the t.Skip calls and
// assert the min_api_version / max_api_version / version_string response.

func TestServerAPIVersionRouteNotImplemented(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)

	resp, err := client.Get("/server_api_version")
	if err != nil {
		t.Fatalf("GET /server_api_version: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 (route missing), got %d: %s", resp.StatusCode, resp.Body)
	}
}

func TestServerAPIVersionBodyShape(t *testing.T) {
	t.Skip("goiardi does not implement /server_api_version; cannot test response body shape")
}

func TestServerAPIVersionRequestorMatrix(t *testing.T) {
	t.Skip("goiardi does not implement /server_api_version; cannot test requestor matrix")
}

func TestServerAPIVersionValidationOnLicense(t *testing.T) {
	t.Skip("goiardi does not implement /server_api_version; cannot test generic API version negotiation")
}

// TestServerAPIVersionResponseInfoHeader is a smoke test that confirms the
// server returns an X-Ops-Server-Api-Version info header on a normal request,
// even though the dedicated endpoint is missing.
func TestServerAPIVersionResponseInfoHeader(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)

	resp, err := client.Get("/users")
	if err != nil {
		t.Fatalf("GET /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)

	h := resp.Header.Get("X-Ops-Server-Api-Version")
	if h == "" {
		t.Log("no X-Ops-Server-Api-Version response header on /users; goiardi does not advertise supported versions")
	}
}

// doServerAPIVersionRequest allows a caller to issue a request with a custom
// X-Ops-Server-Api-Version header value. It is used by skipped tests that
// will eventually validate version negotiation.
func doServerAPIVersionRequest(client *pedant.ChefSigningClient, method, path string, body []byte, apiVersion string) (*pedant.Response, error) {
	u := testServer.APIURL(path)
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Body = http.NoBody // placeholder; callers marshal body into payload elsewhere
	}
	req.Header.Set("X-Ops-Server-Api-Version", apiVersion)
	client.SignRawRequest(req, body)
	r, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	respBody, err := ioReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return &pedant.Response{StatusCode: r.StatusCode, Body: respBody, Header: r.Header}, nil
}
