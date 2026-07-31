package main

import (
	"github.com/ctdk/goiardi/pedant"
	"testing"
)

func TestRolesListEmpty(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/roles")
	if err != nil {
		t.Fatalf("GET /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty role list, got %d entries", len(body))
	}
}

func TestRolesCreateAndRead(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("test_role")
	role := pedant.NewRole(roleName)
	defer client.DeleteOrg("/roles/" + roleName)

	// Create
	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	pedant.AssertBodyContains(t, resp, "/roles/"+roleName)

	// Read
	resp, err = client.GetOrg("/roles/" + roleName)
	if err != nil {
		t.Fatalf("GET /roles/%s: %v", roleName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["name"] != roleName {
		t.Errorf("expected name %q, got %q", roleName, body["name"])
	}
}

func TestRolesCreateDuplicate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("dup_role")
	role := pedant.NewRole(roleName)
	defer client.DeleteOrg("/roles/" + roleName)

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("first POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("second POST /roles: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 409, "already exists")
}

func TestRolesDelete(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("del_role")
	role := pedant.NewRole(roleName)

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.DeleteOrg("/roles/" + roleName)
	if err != nil {
		t.Fatalf("DELETE /roles/%s: %v", roleName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/roles/" + roleName)
	if err != nil {
		t.Fatalf("GET /roles/%s: %v", roleName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestRolesNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/roles/nonexistent_role")
	if err != nil {
		t.Fatalf("GET /roles/nonexistent_role: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestRolesUpdate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("upd_role")
	role := pedant.NewRole(roleName)
	defer client.DeleteOrg("/roles/" + roleName)

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	update := pedant.NewRole(roleName, map[string]interface{}{
		"description": "updated description",
	})
	resp, err = client.PutOrg("/roles/"+roleName, update)
	if err != nil {
		t.Fatalf("PUT /roles/%s: %v", roleName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestRolesJSONClassValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("rl_jsonclass")

	role := pedant.NewRole(roleName)
	role["json_class"] = "Chef::Node"
	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "json_class")
}

func TestRolesChefTypeValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("rl_cheftype")

	role := pedant.NewRole(roleName)
	role["chef_type"] = "node"
	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "chef_type")
}

func TestRolesDefaultAttributes(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("rl_defaults")
	role := pedant.NewRole(roleName)
	defer client.DeleteOrg("/roles/" + roleName)

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/roles/" + roleName)
	if err != nil {
		t.Fatalf("GET /roles/%s: %v", roleName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["default_attributes"] == nil {
		t.Error("expected default_attributes to be set")
	}
	if body["override_attributes"] == nil {
		t.Error("expected override_attributes to be set")
	}
	if body["run_list"] == nil {
		t.Error("expected run_list to be set")
	}
}

func TestRolesEnvRunLists(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("rl_env")
	role := pedant.NewRole(roleName, map[string]interface{}{
		"env_run_lists": map[string]interface{}{
			"prod": []string{"foo", "foo::bar", "recipe[web]", "role[prod]"},
			"dev":  []string{"bar", "recipe[baz]"},
		},
	})
	defer client.DeleteOrg("/roles/" + roleName)

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/roles/" + roleName)
	if err != nil {
		t.Fatalf("GET /roles/%s: %v", roleName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	envRunLists, ok := body["env_run_lists"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected env_run_lists in response, got: %v", body)
	}
	if _, ok := envRunLists["prod"]; !ok {
		t.Errorf("expected 'prod' in env_run_lists")
	}
	if _, ok := envRunLists["dev"]; !ok {
		t.Errorf("expected 'dev' in env_run_lists")
	}
}

func TestRolesRoleNameMismatch(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("rl_mismatch")
	role := pedant.NewRole(roleName)
	defer client.DeleteOrg("/roles/" + roleName)

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Update with wrong name in payload
	update := pedant.NewRole("wrong_name")
	resp, err = client.PutOrg("/roles/"+roleName, update)
	if err != nil {
		t.Fatalf("PUT /roles/%s: %v", roleName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "Role name mismatch")
}

// --- Environment tests ---

func TestRolesInvalidKeys(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("rl_inv_keys")
	role := pedant.NewRole(roleName)
	role["invalid_key"] = "some_value"

	resp, err := client.PostOrg("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 400)
}
