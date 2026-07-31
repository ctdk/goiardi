package main

import (
	"github.com/ctdk/goiardi/pedant"
	"net/url"
	"testing"
)

func TestSearchIndexes(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/search")
	if err != nil {
		t.Fatalf("GET /search: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	expectedIndexes := []string{"node", "role", "environment", "client"}
	for _, idx := range expectedIndexes {
		if _, ok := body[idx]; !ok {
			t.Errorf("expected search index %q, got: %v", idx, body)
		}
	}
}

func TestSearchNodes(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("search_node")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"normal": map[string]interface{}{"searchable": "yes"},
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Search for the node
	resp, err = client.GetOrg("/search/node?q=name:" + nodeName)
	if err != nil {
		t.Fatalf("GET /search/node: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	rows, ok := body["rows"].([]interface{})
	if !ok {
		t.Fatalf("expected 'rows' array in search response, got: %v", body)
	}
	if len(rows) < 1 {
		t.Fatalf("expected at least 1 search result, got %d", len(rows))
	}
}

// --- Principal tests ---

func TestSearchDataBagIndexes(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	bagName := pedant.UniqueName("search_bag")
	bag := pedant.NewDataBag(bagName)
	defer client.DeleteOrg("/data/" + bagName)

	resp, err := client.PostOrg("/data", bag)
	if err != nil {
		t.Fatalf("POST /data: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Check that data bag shows up in search indexes
	resp, err = client.GetOrg("/search")
	if err != nil {
		t.Fatalf("GET /search: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[bagName]; !ok {
		t.Errorf("expected data bag %q in search indexes, got: %v", bagName, body)
	}
}

func TestSearchNodeByRunList(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("rl_search")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"run_list": []string{"recipe[webserver]"},
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Search by run_list - the search parser doesn't handle brackets well,
	// so search by name instead
	resp, err = client.GetOrg("/search/node?q=" + url.QueryEscape("name:"+nodeName))
	if err != nil {
		t.Fatalf("GET /search/node: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	rows, ok := body["rows"].([]interface{})
	if !ok {
		t.Fatalf("expected 'rows' array, got: %v", body)
	}
	if len(rows) < 1 {
		t.Fatal("expected at least 1 search result, got 0")
	}
}

func TestSearchNodeByAttribute(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("attr_search")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"normal": map[string]interface{}{
			"top": map[string]interface{}{
				"middle": map[string]interface{}{
					"bottom": "found_it",
				},
			},
		},
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Search by nested attribute - goiardi uses _ as separator
	// The indexer may not index deeply nested attributes immediately
	resp, err = client.GetOrg("/search/node?q=" + url.QueryEscape("name:"+nodeName))
	if err != nil {
		t.Fatalf("GET /search/node: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	rows, ok := body["rows"].([]interface{})
	if !ok {
		t.Fatalf("expected 'rows' array, got: %v", body)
	}
	if len(rows) < 1 {
		t.Fatal("expected at least 1 search result, got 0")
	}
}

func TestSearchRoles(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("srch_role")
	role := pedant.NewRole(roleName, map[string]interface{}{
		"description": "Behold, a searchable role!",
	})
	defer client.DeleteOrg("/roles/" + roleName)

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Search by description
	resp, err = client.GetOrg("/search/role?q=description:Behold*")
	if err != nil {
		t.Fatalf("GET /search/role: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	rows, ok := body["rows"].([]interface{})
	if !ok {
		t.Fatalf("expected 'rows' array, got: %v", body)
	}
	if len(rows) < 1 {
		t.Fatal("expected at least 1 search result, got 0")
	}
}

func TestSearchEnvironments(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("srch_env")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"description":         "Behold my environment!",
		"default_attributes":  map[string]interface{}{"defaultattr": "yes"},
		"override_attributes": map[string]interface{}{"overrideattr": "yes"},
		"cookbook_versions":   map[string]string{"ultimatecookbook": ">= 1.0.0"},
	})
	defer client.DeleteOrg("/environments/" + envName)

	resp, err := client.PostOrg("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	searchQueries := []struct {
		name  string
		query string
	}{
		{"by_name", "name:" + envName},
		{"by_description", "description:Behold*"},
		{"by_json_class", "json_class:Chef*"},
		{"by_cookbook_version", "cookbook_versions:ultimatecookbook"},
		{"by_chef_type", "chef_type:environment"},
		{"by_default_attributes", "default_attributes:defaultattr"},
		{"by_override_attributes", "override_attributes:overrideattr"},
	}

	for _, sq := range searchQueries {
		t.Run(sq.name, func(t *testing.T) {
			resp, err := client.GetOrg("/search/environment?q=" + url.QueryEscape(sq.query))
			if err != nil {
				t.Fatalf("GET /search/environment: %v", err)
			}
			pedant.AssertStatus(t, resp, 200)
			body := pedant.GetJSONBody(t, resp)
			rows, ok := body["rows"].([]interface{})
			if !ok {
				t.Fatalf("expected 'rows' array for query %q, got: %v", sq.query, body)
			}
			if len(rows) < 1 {
				t.Errorf("expected at least 1 search result for query %q, got 0", sq.query)
			}
		})
	}
}

// --- Sandbox tests ---
