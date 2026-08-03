package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Cookbook Artifacts endpoint tests ---
//
// Ported from oc-chef-pedant:
//   spec/api/cookbook_artifacts/create_spec.rb
//   spec/api/cookbook_artifacts/read_spec.rb
//
// Chef Server "cookbook artifacts" are policyfile-backed cookbook revisions,
// identified by a 40-character hexadecimal identifier (a git blob-like SHA)
// rather than a semver version. They use the same underlying storage as
// regular cookbooks but a different URL namespace
// (/organizations/:org/cookbook_artifacts/:name/:identifier).
//
// Known goiardi gap documented by these tests:
//   * goiardi does NOT implement cookbook artifacts. The default container
//     "cookbook_artifacts" exists, but there is no route handler for
//     /cookbook_artifacts. Any request to the endpoint falls through to the
//     notFoundHandler and returns 404 with body {"error":["not found 12345"]}.
//   * Because the route is missing, the create, list, and read specs from
//     chef-pedant cannot be exercised against real artifact objects. We therefore
//     assert the current 404 behavior and skip the detailed positive tests.
//   * Authorization also cannot be tested: goiardi returns 404 before any
//     permission check. Chef Server would allow admin/superuser create and
//     read, normal user/client read (container ACL defaults), and reject
//     invalid/outside users with 401/403.
//
// When goiardi later adds cookbook artifact support, remove the t.Skip calls
// and implement the positive test matrix below.

const defaultCookbookArtifactID = "1111111111111111111111111111111111111111"

// artifactPayload returns a minimal v0-style cookbook artifact create body.
func artifactPayload(name, identifier string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"identifier":  identifier,
		"version":     "1.0.0",
		"chef_type":   "cookbook_version",
		"frozen?":     false,
		"recipes":     []interface{}{},
		"definitions": []interface{}{},
		"libraries":   []interface{}{},
		"attributes":  []interface{}{},
		"files":       []interface{}{},
		"templates":   []interface{}{},
		"resources":   []interface{}{},
		"providers":   []interface{}{},
		"root_files":  []interface{}{},
		"metadata": map[string]interface{}{
			"version":          "1.0.0",
			"name":             name,
			"maintainer":       "",
			"maintainer_email": "",
			"description":      "",
			"long_description": "",
			"license":          "All rights reserved",
			"dependencies":     map[string]interface{}{},
			"attributes":       map[string]interface{}{},
			"recipes":          map[string]interface{}{},
			"providing":        map[string]interface{}{},
		},
	}
}

// --- Current goiardi behavior: 404 for all cookbook_artifacts paths ---

func TestCookbookArtifactCreateRouteNotImplemented(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	name := pedant.UniqueName("cba")

	resp, err := client.PutOrg("/cookbook_artifacts/"+name+"/"+defaultCookbookArtifactID, artifactPayload(name, defaultCookbookArtifactID))
	if err != nil {
		t.Fatalf("PUT /cookbook_artifacts/%s/%s: %v", name, defaultCookbookArtifactID, err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 (route missing), got %d: %s", resp.StatusCode, resp.Body)
	}
}

func TestCookbookArtifactListRouteNotImplemented(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	resp, err := client.GetOrg("/cookbook_artifacts")
	if err != nil {
		t.Fatalf("GET /cookbook_artifacts: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 (route missing), got %d: %s", resp.StatusCode, resp.Body)
	}
}

func TestCookbookArtifactReadRouteNotImplemented(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	name := pedant.UniqueName("cba")

	resp, err := client.GetOrg("/cookbook_artifacts/" + name + "/" + defaultCookbookArtifactID)
	if err != nil {
		t.Fatalf("GET /cookbook_artifacts/%s/%s: %v", name, defaultCookbookArtifactID, err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 (route missing), got %d: %s", resp.StatusCode, resp.Body)
	}
}

// --- Skipped positive tests that require cookbook artifact support ---

func TestCookbookArtifactCreateBasic(t *testing.T) {
	// Chef Server: PUT /cookbook_artifacts/:name/:identifier returns 201,
	// then GET list shows the artifact and GET by identifier returns it.
	// goiardi has no route, so we cannot test this.
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test create/read positive path")
}

func TestCookbookArtifactCreateRequestorMatrix(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test create requestor matrix")
}

func TestCookbookArtifactCreateValidation(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test create validation")
}

func TestCookbookArtifactCreateConflict(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test create conflict")
}

func TestCookbookArtifactCreateMultipleIdentifiers(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test multiple identifiers per cookbook")
}

func TestCookbookArtifactListRequestorMatrix(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test list requestor matrix")
}

func TestCookbookArtifactListEmpty(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test empty list")
}

func TestCookbookArtifactListMultipleArtifacts(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test list with multiple artifacts")
}

func TestCookbookArtifactReadRequestorMatrix(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test read requestor matrix")
}

func TestCookbookArtifactReadNonExistent(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test read non-existent artifact")
}

func TestCookbookArtifactReadResponseBodyShape(t *testing.T) {
	t.Skip("goiardi does not implement cookbook_artifacts routes; cannot test read response body shape")
}
