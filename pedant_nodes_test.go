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
