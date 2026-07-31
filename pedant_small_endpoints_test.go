package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/pedant"
)

// --- Status (/_status) ---
// oc-chef-pedant status_spec.rb expects GET /_status to return 200 with
// keys: server_version, status, upstreams, keygen, indexing and status "pong".
// goiardi does not implement /_status; this test documents that gap.
func TestStatus(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	resp, err := client.Get("/_status")
	if err != nil {
		t.Fatalf("GET /_status: %v", err)
	}

	// Expectation from chef-server pedant: 200 OK. goiardi currently 404s
	// because /_status is not routed. Failure here is expected and
	// documents the missing endpoint.
	pedant.AssertStatus(t, resp, 200)
	if resp.StatusCode == 200 {
		body := pedant.GetJSONBody(t, resp)
		for _, k := range []string{"status", "upstreams", "keygen", "indexing"} {
			if _, ok := body[k]; !ok {
				t.Errorf("expected key %q in /_status response, got: %v", k, body)
			}
		}
		if body["status"] != "pong" {
			t.Errorf("expected status 'pong', got %v", body["status"])
		}
	}
}

// --- Reindex (/reindex, /organizations/default/search/reindex) ---
// reindex_spec.rb checks that an admin can trigger reindexing. In chef-server
// this shelling out to reindex-opc-organization; goiardi exposes POST
// /reindex and POST /organizations/:org/search/reindex which return {"reindex":"OK"}.
func TestReindex(t *testing.T) {
	admin := testServer.NewClient(testServer.AdminUser)

	// Global reindex requires master reindex:update permission. The admin
	// user is not the pivotal superuser, so this will likely be forbidden
	// for non-superusers. Test both endpoints.
	t.Run("global_reindex_superuser", func(t *testing.T) {
		client := testServer.NewClient(testServer.Superuser)
		resp, err := client.Post("/reindex", nil)
		if err != nil {
			t.Fatalf("POST /reindex: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
		body := pedant.GetJSONBody(t, resp)
		if body["reindex"] != "OK" {
			t.Errorf("expected reindex=OK, got %v", body)
		}
	})

	t.Run("global_reindex_admin_forbidden", func(t *testing.T) {
		resp, err := admin.Post("/reindex", nil)
		if err != nil {
			t.Fatalf("POST /reindex: %v", err)
		}
		// admin is not the pivotal superuser; expect 403.
		pedant.AssertStatus(t, resp, 403)
	})

	t.Run("org_reindex_admin", func(t *testing.T) {
		resp, err := admin.PostOrg("/search/reindex", nil)
		if err != nil {
			t.Fatalf("POST /organizations/default/search/reindex: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
		body := pedant.GetJSONBody(t, resp)
		if body["reindex"] != "OK" {
			t.Errorf("expected reindex=OK, got %v", body)
		}
	})

	t.Run("org_reindex_normal_user_forbidden", func(t *testing.T) {
		client := testServer.NewClient(testServer.NormalUser)
		resp, err := client.PostOrg("/search/reindex", nil)
		if err != nil {
			t.Fatalf("POST /organizations/default/search/reindex: %v", err)
		}
		pedant.AssertStatus(t, resp, 403)
	})
}

// --- Controls (/organizations/default/control) ---
// controls_spec.rb expects GET/POST /controls to return 410 Gone with an error.
func TestControls(t *testing.T) {
	admin := testServer.NewClient(testServer.AdminUser)

	t.Run("post_gone", func(t *testing.T) {
		resp, err := admin.PostOrg("/control", map[string]interface{}{})
		if err != nil {
			t.Fatalf("POST /control: %v", err)
		}
		pedant.AssertStatus(t, resp, 410)
		pedant.AssertBodyContains(t, resp, "error")
	})

	t.Run("get_gone", func(t *testing.T) {
		resp, err := admin.GetOrg("/control")
		if err != nil {
			t.Fatalf("GET /control: %v", err)
		}
		pedant.AssertStatus(t, resp, 410)
		pedant.AssertBodyContains(t, resp, "error")
	})
}

// --- Headers ---
// header_spec.rb verifies X-Chef-Version handling and X-Forwarded-* headers.
// chef-server accepts high versions (999.0.0) and rejects low ones (9.0.1).
// goiardi does not currently enforce X-Chef-Version, so these tests document
// the gap.
func TestHeader(t *testing.T) {
	admin := testServer.NewClient(testServer.AdminUser)

	makeReq := func(extraHeaders map[string]string) (*pedant.Response, error) {
		u := testServer.OrgURL("/users")
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		// Sign with admin credentials
		admin.SignRawRequest(req, nil)
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}
		resp, err := admin.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := readAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return &pedant.Response{StatusCode: resp.StatusCode, Body: body, Header: resp.Header}, nil
	}

	t.Run("high_version_accepted", func(t *testing.T) {
		resp, err := makeReq(map[string]string{"X-Chef-Version": "999.0.0"})
		if err != nil {
			t.Fatalf("GET /users with high X-Chef-Version: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
	})

	t.Run("low_version_rejected", func(t *testing.T) {
		resp, err := makeReq(map[string]string{"X-Chef-Version": "9.0.1"})
		if err != nil {
			t.Fatalf("GET /users with low X-Chef-Version: %v", err)
		}
		// chef-server returns 400 for versions < 10. goiardi ignores the
		// header currently, so this failure documents the gap.
		pedant.AssertStatus(t, resp, 400)
	})

	for _, h := range []string{"Host", "For", "Server"} {
		name := "x_forwarded_" + strings.ToLower(h)
		t.Run(name, func(t *testing.T) {
			resp, err := makeReq(map[string]string{fmt.Sprintf("X-Forwarded-%s", h): "abc:443,def"})
			if err != nil {
				t.Fatalf("GET /users with X-Forwarded-%s: %v", h, err)
			}
			pedant.AssertStatus(t, resp, 200)
		})
	}
}

// --- License (/license) ---
// license_spec.rb tests GET /license with 0, 1, 25, and 26 nodes. chef-server
// counts nodes and reports limit_exceeded. goiardi returns a static unlimited
// license (node_license=1e9, limit_exceeded=false).
func TestLicense(t *testing.T) {
	admin := testServer.NewClient(testServer.AdminUser)

	t.Run("no_nodes", func(t *testing.T) {
		resp, err := admin.Get("/license")
		if err != nil {
			t.Fatalf("GET /license: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
		body := pedant.GetJSONBody(t, resp)

		// goiardi static response
		if body["limit_exceeded"] != false {
			t.Errorf("expected limit_exceeded=false, got %v", body["limit_exceeded"])
		}
		if body["node_count"] != 0.0 {
			t.Errorf("expected node_count=0, got %v", body["node_count"])
		}
	})

	t.Run("with_nodes", func(t *testing.T) {
		nodeName := pedant.UniqueName("license_node")
		node := pedant.NewNode(nodeName)
		defer admin.DeleteOrg("/nodes/" + nodeName)

		resp, err := admin.PostOrg("/nodes", node)
		if err != nil {
			t.Fatalf("POST /nodes: %v", err)
		}
		pedant.AssertStatus(t, resp, 201)

		resp, err = admin.Get("/license")
		if err != nil {
			t.Fatalf("GET /license: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
		body := pedant.GetJSONBody(t, resp)

		// chef-server would count 1 node. goiardi always reports 0.
		// This failure documents the gap.
		if body["node_count"] != 1.0 {
			t.Errorf("expected node_count=1, got %v (goiardi does not count nodes for license)", body["node_count"])
		}
		if body["limit_exceeded"] != false {
			t.Errorf("expected limit_exceeded=false, got %v", body["limit_exceeded"])
		}
	})

	t.Run("unauthenticated_client_rejected", func(t *testing.T) {
		// Request with an unknown client name; goiardi auth will reject.
		bogus := &pedant.TestRequestor{
			Name:       "invalid_user",
			PrivateKey: testServer.AdminUser.PrivateKey,
		}
		client := testServer.NewClient(bogus)
		resp, err := client.Get("/license")
		if err != nil {
			t.Fatalf("GET /license as invalid user: %v", err)
		}
		pedant.AssertStatus(t, resp, 401)
	})

	t.Run("client_rejected", func(t *testing.T) {
		client := testServer.NewClient(testServer.NormalClient)
		resp, err := client.Get("/license")
		if err != nil {
			t.Fatalf("GET /license as client: %v", err)
		}
		// chef-server returns 401 for clients on /license.
		pedant.AssertStatus(t, resp, 401)
	})
}

// --- Groups ACL ---
// groups_acl_spec.rb tests removing the admins/clients/users group from the
// grant ACE (should be forbidden for non-superuser, OK for superuser) and from
// the read ACE (OK for admin).
func TestGroupsACL(t *testing.T) {
	admin := testServer.NewClient(testServer.AdminUser)
	superuser := testServer.NewClient(testServer.Superuser)

	for _, groupName := range []string{"admins", "clients", "users"} {
		t.Run(groupName+"_grant_superuser_ok", func(t *testing.T) {
			resp, err := superuser.PutOrg("/groups/"+groupName+"/_acl/grant", map[string]interface{}{
				"grant": map[string]interface{}{
					"actors": []string{"pivotal"},
					"groups": []string{},
				},
			})
			if err != nil {
				t.Fatalf("PUT /groups/%s/_acl/grant as superuser: %v", groupName, err)
			}
			pedant.AssertStatus(t, resp, 200)
		})

		t.Run(groupName+"_grant_admin_forbidden", func(t *testing.T) {
			resp, err := admin.PutOrg("/groups/"+groupName+"/_acl/grant", map[string]interface{}{
				"grant": map[string]interface{}{
					"actors": []string{"pivotal"},
					"groups": []string{},
				},
			})
			if err != nil {
				t.Fatalf("PUT /groups/%s/_acl/grant as admin: %v", groupName, err)
			}
			pedant.AssertStatus(t, resp, 403)
		})

		t.Run(groupName+"_read_admin_ok", func(t *testing.T) {
			resp, err := admin.PutOrg("/groups/"+groupName+"/_acl/read", map[string]interface{}{
				"read": map[string]interface{}{
					"actors": []string{"pivotal"},
					"groups": []string{},
				},
			})
			if err != nil {
				t.Fatalf("PUT /groups/%s/_acl/read as admin: %v", groupName, err)
			}
			pedant.AssertStatus(t, resp, 200)
		})
	}
}

// --- Validate (/validate) ---
// validate_spec.rb exercises an internal /validate endpoint that chef-server
// uses to verify signed requests. goiardi does not implement /validate.
func TestValidate(t *testing.T) {
	admin := testServer.NewClient(testServer.AdminUser)

	u := testServer.APIURL("/validate/organizations/default/nodes")
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatalf("building validate request: %v", err)
	}
	admin.SignRawRequest(req, nil)

	resp, err := admin.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("GET /validate/...: %v", err)
	}
	defer resp.Body.Close()
	body, err := readAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response: %v", err)
	}

	// chef-server returns 200 and requestor info. goiardi returns 404
	// because /validate is not routed. Failure documents the gap.
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 for /validate, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

// --- Universe (/organizations/default/universe) ---
// universe_spec.rb expects GET /universe to return a hash of cookbooks with
// versions, dependencies, location_path and location_type.
func TestUniverse(t *testing.T) {
	admin := testServer.NewClient(testServer.AdminUser)

	t.Run("empty_universe", func(t *testing.T) {
		resp, err := admin.GetOrg("/universe")
		if err != nil {
			t.Fatalf("GET /universe: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
		body := pedant.GetJSONBody(t, resp)
		if len(body) != 0 {
			t.Errorf("expected empty universe, got %v", body)
		}
	})

	t.Run("with_cookbooks", func(t *testing.T) {
		fooName := pedant.UniqueName("foo")
		barName := pedant.UniqueName("bar")
		defer admin.DeleteOrg("/cookbooks/" + fooName + "/1.2.3")
		defer admin.DeleteOrg("/cookbooks/" + barName + "/1.2.3")

		foo := pedant.NewCookbook(fooName, "1.2.3", map[string]interface{}{
			"metadata": map[string]interface{}{
				"version":      "1.2.3",
				"name":         fooName,
				"dependencies": map[string]interface{}{"bar": ">= 1.1.1"},
			},
		})
		bar := pedant.NewCookbook(barName, "1.2.3", map[string]interface{}{
			"metadata": map[string]interface{}{
				"version":      "1.2.3",
				"name":         barName,
				"dependencies": map[string]interface{}{},
			},
		})

		resp, err := admin.PutOrg("/cookbooks/"+fooName+"/1.2.3", foo)
		if err != nil {
			t.Fatalf("PUT cookbook foo: %v", err)
		}
		pedant.AssertStatus(t, resp, 201)

		resp, err = admin.PutOrg("/cookbooks/"+barName+"/1.2.3", bar)
		if err != nil {
			t.Fatalf("PUT cookbook bar: %v", err)
		}
		pedant.AssertStatus(t, resp, 201)

		resp, err = admin.GetOrg("/universe")
		if err != nil {
			t.Fatalf("GET /universe: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
		body := pedant.GetJSONBody(t, resp)

		fooEntry, ok := body[fooName].(map[string]interface{})
		if !ok {
			t.Fatalf("expected cookbook %q in universe, got %v", fooName, body)
		}
		ver, ok := fooEntry["1.2.3"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected version 1.2.3 for %q, got %v", fooName, fooEntry)
		}
		if ver["location_type"] != "chef_server" {
			t.Errorf("expected location_type 'chef_server', got %v", ver["location_type"])
		}
		deps, ok := ver["dependencies"].(map[string]interface{})
		if !ok || deps["bar"] != ">= 1.1.1" {
			t.Errorf("expected dependencies bar '>= 1.1.1', got %v", ver["dependencies"])
		}
		if !strings.Contains(ver["location_path"].(string), "/cookbooks/"+fooName+"/1.2.3") {
			t.Errorf("expected location_path to contain /cookbooks/%s/1.2.3, got %v", fooName, ver["location_path"])
		}
	})
}

// --- Pedant self-diagnostic / platform tests ---
// pedant_spec.rb is mostly internal matcher and shared()/let() tests, plus a
// role-util integration test. We port the spirit: verify that matchers/helpers
// behave and that basic platform objects exist.
func TestPedant(t *testing.T) {
	t.Run("requestors_exist", func(t *testing.T) {
		for _, r := range []*pedant.TestRequestor{
			testServer.AdminUser,
			testServer.NormalUser,
			testServer.AdminClient,
			testServer.NormalClient,
			testServer.ValidatorClient,
			testServer.OutsideUser,
			testServer.Superuser,
		} {
			if r == nil || r.Name == "" {
				t.Errorf("expected requestor to be configured, got %+v", r)
			}
		}
	})

	t.Run("server_base_url_configured", func(t *testing.T) {
		if config.Config.Hostname == "" {
			t.Error("expected config hostname to be set")
		}
	})

	t.Run("list_users_returns_users", func(t *testing.T) {
		admin := testServer.NewClient(testServer.AdminUser)
		resp, err := admin.Get("/users")
		if err != nil {
			t.Fatalf("GET /users: %v", err)
		}
		pedant.AssertStatus(t, resp, 200)
		body := pedant.GetJSONBody(t, resp)
		// Should contain at least the pivotal superuser.
		if _, ok := body[config.SuperuserName]; !ok {
			t.Errorf("expected %q in user list, got %v", config.SuperuserName, body)
		}
	})

	t.Run("strictly_match_helper", func(t *testing.T) {
		a := map[string]interface{}{"a": 1, "b": 2}
		b := map[string]interface{}{"b": 2, "a": 1}
		if !mapsEqual(a, b) {
			t.Errorf("expected %v to strictly_match %v", a, b)
		}
		c := map[string]interface{}{"b": 3, "a": 1}
		if mapsEqual(a, c) {
			t.Errorf("expected %v to NOT strictly_match %v", a, c)
		}
	})

	t.Run("loosely_match_helper", func(t *testing.T) {
		a := map[string]interface{}{"a": 1, "b": 2}
		b := map[string]interface{}{"a": 1}
		if !mapContains(a, b) {
			t.Errorf("expected %v to loosely_match %v", a, b)
		}
		c := map[string]interface{}{"a": 3}
		if mapContains(a, c) {
			t.Errorf("expected %v to NOT loosely_match %v", a, c)
		}
	})
}

// --- local helpers ---

func readAll(r interface{}) ([]byte, error) {
	// generic helper for reading an io.ReadCloser; kept minimal.
	switch rc := r.(type) {
	case *http.Response:
		return ioReadAll(rc.Body)
	default:
		return ioReadAll(r)
	}
}

func ioReadAll(r interface{}) ([]byte, error) {
	if rc, ok := r.(interface{ Read([]byte) (int, error) }); ok {
		var buf bytes.Buffer
		_, err := buf.ReadFrom(rc.(interface {
			Read([]byte) (int, error)
		}))
		return buf.Bytes(), err
	}
	return nil, fmt.Errorf("unsupported reader type")
}

func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		avj, _ := json.Marshal(v)
		bvj, _ := json.Marshal(bv)
		if !bytes.Equal(avj, bvj) {
			return false
		}
	}
	return true
}

func mapContains(target, spec map[string]interface{}) bool {
	for k, v := range spec {
		tv, ok := target[k]
		if !ok {
			return false
		}
		tvj, _ := json.Marshal(tv)
		vj, _ := json.Marshal(v)
		if !bytes.Equal(tvj, vj) {
			return false
		}
	}
	return true
}
