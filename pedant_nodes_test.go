package main

import (
	"fmt"
	"github.com/ctdk/goiardi/pedant"
	"testing"
)

func TestNodesListEmpty(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/nodes")
	if err != nil {
		t.Fatalf("GET /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty node list, got %d entries", len(body))
	}
}

func TestNodesCreateAndRead(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("test_node")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	// Create
	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	pedant.AssertBodyContains(t, resp, "/nodes/"+nodeName)

	// Read
	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["name"] != nodeName {
		t.Errorf("expected name %q, got %q", nodeName, body["name"])
	}
}

func TestNodesCreateDuplicate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("dup_node")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	// Create first
	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("first POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Create duplicate
	resp, err = client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("second POST /nodes: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 409, "Node already exists")
}

func TestNodesDelete(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("del_node")
	node := pedant.NewNode(nodeName)

	// Create
	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Delete
	resp, err = client.DeleteOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("DELETE /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify gone
	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestNodesNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/nodes/nonexistent_node")
	if err != nil {
		t.Fatalf("GET /nodes/nonexistent_node: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestNodesUpdate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("upd_node")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	// Create
	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Update
	update := map[string]interface{}{
		"name":       nodeName,
		"json_class": "Chef::Node",
		"chef_type":  "node",
		"normal":     map[string]interface{}{"updated": "yes"},
		"run_list":   []string{},
	}
	resp, err = client.PutOrg("/nodes/"+nodeName, update)
	if err != nil {
		t.Fatalf("PUT /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify
	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	normal := body["normal"].(map[string]interface{})
	if normal["updated"] != "yes" {
		t.Errorf("expected normal.updated = 'yes', got %v", normal["updated"])
	}
}

func TestNodesJSONClassValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("jsonclass_test")

	// Valid json_class
	node := pedant.NewNode(nodeName)
	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	client.DeleteOrg("/nodes/" + nodeName)

	// Invalid json_class
	nodeName2 := pedant.UniqueName("jsonclass_bad")
	node2 := pedant.NewNode(nodeName2)
	node2["json_class"] = "Chef::Role"
	resp, err = client.PostOrg("/nodes", node2)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "json_class")
}

func TestNodesDefaultAttributes(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("default_attr")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	// Create without default attributes
	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Verify defaults
	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["chef_environment"] != "_default" {
		t.Errorf("expected chef_environment '_default', got %v", body["chef_environment"])
	}
}

func TestNodesListAfterCreate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("list_test")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	// Create
	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// List
	resp, err = client.GetOrg("/nodes")
	if err != nil {
		t.Fatalf("GET /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[nodeName]; !ok {
		t.Errorf("expected node %q in list, got: %v", nodeName, body)
	}
}

// --- Role tests ---

func TestNodesEnvironmentScopedList(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("env_node")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"chef_environment": "_default",
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// List nodes in _default environment
	resp, err = client.GetOrg("/environments/_default/nodes")
	if err != nil {
		t.Fatalf("GET /environments/_default/nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[nodeName]; !ok {
		t.Errorf("expected node %q in _default environment, got: %v", nodeName, body)
	}
}

func TestNodesChefEnvironmentValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	tests := []struct {
		name  string
		env   string
		valid bool
	}{
		{"valid_env", "PEDANT", true},
		{"valid_env2", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqurstuvwxyz0123456789-_", true},
		{"no_colon", "pedant:no_colon_in_environment_name", false},
		{"no_at", "pedant@127.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := pedant.NewNode(tt.name, map[string]interface{}{
				"chef_environment": tt.env,
			})
			resp, err := client.PostOrg("/nodes", node)
			if err != nil {
				t.Fatalf("POST /nodes: %v", err)
			}
			if tt.valid {
				pedant.AssertStatus(t, resp, 201)
				client.DeleteOrg("/nodes/" + tt.name)
			} else {
				pedant.AssertStatus(t, resp, 400)
			}
		})
	}
}

func TestNodesUpdateNameMismatch(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("nmm_node")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Update with wrong name in payload
	update := pedant.NewNode("wrong_name", map[string]interface{}{
		"run_list": []string{},
	})
	resp, err = client.PutOrg("/nodes/"+nodeName, update)
	if err != nil {
		t.Fatalf("PUT /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "Node name mismatch")
}

func TestNodesMultipleCreation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	count := 7
	nodeNames := make([]string, count)
	for i := 0; i < count; i++ {
		nodeNames[i] = pedant.UniqueName(fmt.Sprintf("multi_node_%d", i))
		node := pedant.NewNode(nodeNames[i])
		resp, err := client.PostOrg("/nodes", node)
		if err != nil {
			t.Fatalf("POST /nodes %s: %v", nodeNames[i], err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, name := range nodeNames {
			client.DeleteOrg("/nodes/" + name)
		}
	}()

	resp, err := client.GetOrg("/nodes")
	if err != nil {
		t.Fatalf("GET /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	// Verify our nodes show up
	for _, name := range nodeNames {
		if _, ok := body[name]; !ok {
			t.Errorf("expected node %q in list, not found", name)
		}
	}
}

// --- More Environment tests ---

func TestNodesAttributeTypeValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	nodeName := pedant.UniqueName("attr_type")

	invalidAttrs := []struct {
		field string
		value interface{}
	}{
		{"normal", "string"},
		{"normal", 123},
		{"default", "string"},
		{"override", "string"},
		{"automatic", "string"},
	}

	for _, ia := range invalidAttrs {
		t.Run(ia.field, func(t *testing.T) {
			node := pedant.NewNode(nodeName, map[string]interface{}{
				ia.field: ia.value,
			})
			resp, err := client.PostOrg("/nodes", node)
			if err != nil {
				t.Fatalf("POST /nodes: %v", err)
			}
			pedant.AssertStatus(t, resp, 400)
		})
	}
}

func TestNodesInvalidKeys(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("inv_keys")
	node := pedant.NewNode(nodeName)
	node["invalid_key"] = "some_value"

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 400)
}

// --- Requestor matrix ---

// requestorMatrixForNodes verifies basic CRUD authorization for the given
// requestor. In goiardi, authorization differs from Chef Server: there is no
// webui-based pivotal user; admin/superuser collapse to the configured
// superuser. Normal users associated with the default org have broad read/write
// access to nodes. Normal clients associated with the default org cannot read
// or write nodes. Outside users/clients are unauthenticated (401) and cannot
// act as a node requestor.
func requestorMatrixForNodes(t *testing.T, req *pedant.TestRequestor, canRead, canWrite bool) {
	admin := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("matrix_node")
	node := pedant.NewNode(nodeName)
	defer admin.DeleteOrg("/nodes/" + nodeName)

	resp, err := admin.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	client := testServer.NewClient(req)

	// Read
	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	if canRead {
		pedant.AssertStatus(t, resp, 200)
	} else {
		pedant.AssertStatus(t, resp, 403)
	}

	// Update
	update := pedant.NewNode(nodeName, map[string]interface{}{
		"normal": map[string]interface{}{"matrix": true},
	})
	resp, err = client.PutOrg("/nodes/"+nodeName, update)
	if err != nil {
		t.Fatalf("PUT /nodes/%s: %v", nodeName, err)
	}
	if canWrite {
		pedant.AssertStatus(t, resp, 200)
	} else {
		pedant.AssertStatus(t, resp, 403)
	}

	// Delete and recreate as admin for the next requestor if needed.
	resp, err = admin.DeleteOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("DELETE /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Create with the requestor under test.
	resp, err = client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	if canWrite {
		pedant.AssertStatus(t, resp, 201)
	} else {
		pedant.AssertStatus(t, resp, 403)
	}
	if resp.StatusCode == 201 {
		defer admin.DeleteOrg("/nodes/" + nodeName)
	}
}

func TestNodesRequestorMatrixSuperuser(t *testing.T) {
	requestorMatrixForNodes(t, testServer.Superuser, true, true)
}

func TestNodesRequestorMatrixAdminUser(t *testing.T) {
	requestorMatrixForNodes(t, testServer.AdminUser, true, true)
}

func TestNodesRequestorMatrixNormalUser(t *testing.T) {
	requestorMatrixForNodes(t, testServer.NormalUser, true, true)
}

// goiardi divergence: normal clients associated with the default org cannot
// read or write nodes; Chef Server ACLs may allow it.
func TestNodesRequestorMatrixNormalClient(t *testing.T) {
	requestorMatrixForNodes(t, testServer.NormalClient, false, false)
}

// goiardi divergence: outside users are unknown actors and are rejected with
// 401 rather than 403.
func TestNodesRequestorMatrixOutsideUser(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)
	nodeName := pedant.UniqueName("outside_node")
	resp, err := client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 401)
}

// --- PUT create-or-update semantics ---

// NOTE: goiardi does not implement Chef Server's PUT-create semantics for
// nodes. The Ruby pedant spec uses PUT to create nodes; goiardi requires a
// POST to /nodes to create a node first. We test PUT against an existing node.

func TestNodesPutCreatesNewNode(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("put_create_node")
	update := pedant.NewNode(nodeName, map[string]interface{}{
		"run_list": []string{},
	})

	resp, err := client.PutOrg("/nodes/"+nodeName, update)
	if err != nil {
		t.Fatalf("PUT /nodes/%s: %v", nodeName, err)
	}
	// goiardi divergence: PUT to a nonexistent node currently returns 404
	// rather than creating it. Documented in goiardi gaps.
	pedant.AssertStatus(t, resp, 404)
}

func TestNodesPutUpdatesExistingNode(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("put_update_node")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	update := map[string]interface{}{
		"name":             nodeName,
		"json_class":       "Chef::Node",
		"chef_type":        "node",
		"chef_environment": "_default",
		"default":          map[string]interface{}{"foo": "bar"},
		"normal":           map[string]interface{}{},
		"override":         map[string]interface{}{},
		"automatic":        map[string]interface{}{},
		"run_list":         []string{},
	}
	resp, err = client.PutOrg("/nodes/"+nodeName, update)
	if err != nil {
		t.Fatalf("PUT /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	def := body["default"].(map[string]interface{})
	if def["foo"] != "bar" {
		t.Errorf("expected default.foo = 'bar', got %v", def["foo"])
	}
}

func TestNodesPutNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("put_missing_node")
	update := map[string]interface{}{
		"json_class": "Chef::Node",
		"run_list":   []string{},
	}
	resp, err := client.PutOrg("/nodes/"+nodeName, update)
	if err != nil {
		t.Fatalf("PUT /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

// --- DELETE matrix ---

func TestNodesDeleteNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("delete_missing_node")
	resp, err := client.DeleteOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("DELETE /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestNodesDeleteRequestorMatrix(t *testing.T) {
	cases := []struct {
		req      *pedant.TestRequestor
		canWrite bool
	}{
		{testServer.Superuser, true},
		{testServer.AdminUser, true},
		{testServer.NormalUser, true},
		// goiardi divergence: normal clients associated with the default org
		// cannot delete nodes even though they can read them.
		{testServer.NormalClient, false},
		// goiardi divergence: outside users are unknown actors and are rejected
		// with 401 rather than 403.
		{testServer.OutsideUser, false},
	}
	for _, tc := range cases {
		t.Run(tc.req.Name, func(t *testing.T) {
			client := testServer.NewClient(testServer.AdminUser)
			nodeName := pedant.UniqueName("delete_matrix_node")
			node := pedant.NewNode(nodeName)
			resp, err := client.PostOrg("/nodes", node)
			if err != nil {
				t.Fatalf("POST /nodes: %v", err)
			}
			pedant.AssertStatus(t, resp, 201)

			cc := testServer.NewClient(tc.req)
			resp, err = cc.DeleteOrg("/nodes/" + nodeName)
			if err != nil {
				t.Fatalf("DELETE /nodes/%s: %v", nodeName, err)
			}
			if tc.req == testServer.OutsideUser {
				pedant.AssertStatus(t, resp, 401)
			} else if tc.canWrite {
				pedant.AssertStatus(t, resp, 200)
			} else {
				pedant.AssertStatus(t, resp, 403)
			}
		})
	}
}

func TestNodesDeleteOutsideUser(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)
	nodeName := pedant.UniqueName("delete_outside_node")
	resp, err := client.DeleteOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("DELETE /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 401)
}

// --- Name validation ---

func TestNodesNameValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cases := []struct {
		name  string
		valid bool
	}{
		{"pedant_node", true},
		{"PEDANT_NODE", true},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqurstuvwxyz0123456789-_:", true},
		{"node@127.0.0.1", false},
		{"this has spaces", false},
		{"node@bad", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a unique prefix to avoid collisions with tests that may
			// leave state behind.
			nodeName := pedant.UniqueName("nv")
			node := pedant.NewNode(nodeName)
			node["name"] = tc.name
			resp, err := client.PostOrg("/nodes", node)
			if err != nil {
				t.Fatalf("POST /nodes: %v", err)
			}
			if tc.valid {
				pedant.AssertStatus(t, resp, 201)
				client.DeleteOrg("/nodes/" + tc.name)
			} else {
				pedant.AssertStatus(t, resp, 400)
			}
		})
	}
}

func TestNodesNameConflict(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("conflict_node")
	node := pedant.NewNode(nodeName)
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("first POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Different payload with same name should conflict.
	node2 := pedant.NewNode(nodeName, map[string]interface{}{
		"chef_environment": "_default",
	})
	resp, err = client.PostOrg("/nodes", node2)
	if err != nil {
		t.Fatalf("second POST /nodes: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 409, "Node already exists")
}

// --- Environment scoping ---

func TestNodesEnvironmentScoped(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("env_scoped_node")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"chef_environment": "PEDANT_ENV",
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Chef Server strictly validates that environments exist before assigning a
	// node to them; goiardi accepts the environment name without checking
	// existence, which is a divergence from the Ruby pedant suite.
	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["chef_environment"] != "PEDANT_ENV" {
		t.Errorf("expected chef_environment 'PEDANT_ENV', got %v", body["chef_environment"])
	}
}

func TestNodesListByEnvironment(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("env_list_node")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"chef_environment": "_default",
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/environments/_default/nodes")
	if err != nil {
		t.Fatalf("GET /environments/_default/nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[nodeName]; !ok {
		t.Errorf("expected node %q in _default environment list, got %v", nodeName, body)
	}
}

// --- Run list handling ---

func TestNodesRunList(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("runlist_node")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"run_list": []string{"recipe[foo]", "role[base]", "bar::baz@1.0.0"},
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	runList, ok := body["run_list"].([]interface{})
	if !ok {
		t.Fatalf("expected run_list array, got %T", body["run_list"])
	}
	if len(runList) != 3 {
		t.Errorf("expected 3 run_list items, got %d", len(runList))
	}
}

func TestNodesRunListValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("runlist_bad")

	invalid := []interface{}{
		[]string{"recipe["},
		[]interface{}{123},
		[]string{"this is not valid"},
	}
	for _, rl := range invalid {
		t.Run(fmt.Sprintf("%v", rl), func(t *testing.T) {
			node := pedant.NewNode(nodeName, map[string]interface{}{
				"run_list": rl,
			})
			resp, err := client.PostOrg("/nodes", node)
			if err != nil {
				t.Fatalf("POST /nodes: %v", err)
			}
			pedant.AssertStatus(t, resp, 400)
		})
	}
}

// --- Attribute handling ---

func TestNodesAutomaticAttributes(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("auto_attr")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"automatic": map[string]interface{}{
			"ipaddress": "127.0.0.1",
			"platform":  "ubuntu",
		},
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	auto := body["automatic"].(map[string]interface{})
	if auto["ipaddress"] != "127.0.0.1" {
		t.Errorf("expected automatic.ipaddress = '127.0.0.1', got %v", auto["ipaddress"])
	}
}

func TestNodesAllAttributeLevels(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("all_attrs")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"default":   map[string]interface{}{"level": "default"},
		"normal":    map[string]interface{}{"level": "normal"},
		"override":  map[string]interface{}{"level": "override"},
		"automatic": map[string]interface{}{"level": "automatic"},
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	for _, level := range []string{"default", "normal", "override", "automatic"} {
		m := body[level].(map[string]interface{})
		if m["level"] != level {
			t.Errorf("expected %s.level = %q, got %v", level, level, m["level"])
		}
	}
}

// --- Response body shape ---

func TestNodesResponseShape(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	nodeName := pedant.UniqueName("shape_node")
	node := pedant.NewNode(nodeName, map[string]interface{}{
		"chef_environment": "_default",
		"run_list":         []string{"recipe[web]"},
		"normal":           map[string]interface{}{"foo": "bar"},
	})
	defer client.DeleteOrg("/nodes/" + nodeName)

	resp, err := client.PostOrg("/nodes", node)
	if err != nil {
		t.Fatalf("POST /nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/nodes/" + nodeName)
	if err != nil {
		t.Fatalf("GET /nodes/%s: %v", nodeName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	required := []string{"name", "chef_environment", "run_list", "normal", "override", "automatic", "default"}
	for _, key := range required {
		if _, ok := body[key]; !ok {
			t.Errorf("expected response key %q missing", key)
		}
	}
	if body["name"] != nodeName {
		t.Errorf("expected name %q, got %v", nodeName, body["name"])
	}
	if body["chef_environment"] != "_default" {
		t.Errorf("expected chef_environment '_default', got %v", body["chef_environment"])
	}
}

// --- Bulk node operations ---

// NOTE: goiardi does not implement Chef Server's /nodes/_bulk (or similar bulk
// API). That endpoint is not covered here; it is listed as a divergence.

// --- goiardi divergences ---

// This comment documents known differences between the Ruby chef-pedant
// complete_endpoint_spec.rb for nodes and goiardi's behavior, as discovered
// while porting this chunk:
//
// 1. PUT /nodes/<name> on a nonexistent node returns 404 instead of creating
//    the node. Chef Server treats PUT as create-or-update.
// 2. goiardi does not validate that a node's chef_environment actually exists.
//    It accepts arbitrary environment names.
// 3. goiardi does not implement policy_name/policy_group validation; those
//    fields are accepted if present but are not enforced.
// 4. Bulk node APIs are not implemented.
