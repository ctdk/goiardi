package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Required Recipe (/organizations/:org/required_recipe) ---
//
// Ported from oc-chef-pedant:
//   spec/api/required_recipe_spec.rb
//
// Chef Server's required_recipe endpoint is used by nodes to fetch a
// centrally-managed required recipe. The Ruby spec has two modes:
//   * When required_recipe_enabled is true:
//       - POST returns 405 (Method Not Allowed)
//       - GET with a valid client returns 200
//       - GET with an invalid client returns 401
//       - GET without a valid request returns 400
//   * When required_recipe_enabled is false, all requests return 404.
//
// Known goiardi gap documented by these tests:
//   * goiardi does NOT implement /organizations/:org/required_recipe. For
//     authenticated requestors the request falls through to the
//     notFoundHandler and returns 404. Unauthenticated requestors are rejected
//     with 401 before the route is reached.
//   * Because the endpoint is missing, the requestor matrix and response
//     body shape (a JSON required_recipe object) cannot be exercised.
//
// When goiardi implements required_recipe, remove the t.Skip calls and
// implement the positive and negative cases from the Ruby spec.

func TestRequiredRecipeRouteNotImplemented(t *testing.T) {
	client := testServer.NewClient(testServer.NormalClient)

	resp, err := client.GetOrg("/required_recipe")
	if err != nil {
		t.Fatalf("GET /organizations/default/required_recipe: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 (route missing), got %d: %s", resp.StatusCode, resp.Body)
	}
}

func TestRequiredRecipeGetRequestorMatrix(t *testing.T) {
	t.Skip("goiardi does not implement /required_recipe; cannot test GET requestor matrix")
}

func TestRequiredRecipePostMethodNotAllowed(t *testing.T) {
	t.Skip("goiardi does not implement /required_recipe; cannot test POST 405 behavior")
}

func TestRequiredRecipeResponseBodyShape(t *testing.T) {
	t.Skip("goiardi does not implement /required_recipe; cannot test response body shape")
}

// TestRequiredRecipeDisabledDocumentsCurrentBehavior issues requests in the
// style of the Ruby "required_recipe is disabled" suite. goiardi currently
// behaves like the disabled branch for authenticated requestors: the route is
// missing so it returns 404. Unauthenticated requestors are rejected with 401
// before the route is reached; that difference is documented and accepted.
func TestRequiredRecipeDisabledDocumentsCurrentBehavior(t *testing.T) {
	cases := []struct {
		name           string
		requestor      *pedant.TestRequestor
		method         string
		expectedStatus int
	}{
		{"valid_client_get", testServer.NormalClient, "GET", 404},
		{"invalid_client_get", bogusRequestor(), "GET", 401},
		{"valid_client_post", testServer.NormalClient, "POST", 404},
		{"invalid_client_post", bogusRequestor(), "POST", 401},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := testServer.NewClient(tc.requestor)
			var resp *pedant.Response
			var err error
			switch tc.method {
			case "GET":
				resp, err = client.GetOrg("/required_recipe")
			case "POST":
				resp, err = client.PostOrg("/required_recipe", map[string]interface{}{})
			default:
				t.Fatalf("unsupported method %q", tc.method)
			}
			if err != nil {
				t.Fatalf("%s /organizations/default/required_recipe as %s: %v", tc.method, tc.requestor.Name, err)
			}
			// goiardi authenticates before routing. Valid clients hit the
			// missing route and get 404; invalid clients fail auth first.
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d: %s", tc.expectedStatus, resp.StatusCode, resp.Body)
			}
		})
	}
}
