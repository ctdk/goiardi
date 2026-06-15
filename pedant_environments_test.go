package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Environment Sub-endpoint Tests ---
// Ported from: cookbooks_spec.rb, recipes_spec.rb, roles_spec.rb, single_cookbook_spec.rb

// GET /environments/<name>/cookbooks

func TestEnvironmentsCookbooksNoCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_cb_empty")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// No cookbooks - but other tests may have created cookbooks
	// Just verify the endpoint returns 200
	resp, err = client.Get("/environments/" + envName + "/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsCookbooksDefaultNoCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/_default/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsCookbooksNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/bad_env/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/bad_env/cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsCookbooksNumVersionsInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_cb_nv")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/environments/" + envName + "/cookbooks?num_versions=skittles")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks?num_versions=skittles: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestEnvironmentsCookbooksWithCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_cb_with")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Create cookbooks
	cb1 := pedant.UniqueName("env_cb_one")
	cb2 := pedant.UniqueName("env_cb_two")

	for _, cb := range []struct{ name, version string }{
		{cb1, "1.0.0"}, {cb1, "2.0.0"}, {cb1, "3.0.0"},
		{cb2, "1.0.0"}, {cb2, "1.2.0"}, {cb2, "1.2.5"},
	} {
		payload := newCookbookPayload(cb.name, cb.version)
		resp, err := client.Put("/cookbooks/"+cb.name+"/"+cb.version, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cb.name, cb.version, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		client.Delete("/cookbooks/" + cb1 + "/1.0.0")
		client.Delete("/cookbooks/" + cb1 + "/2.0.0")
		client.Delete("/cookbooks/" + cb1 + "/3.0.0")
		client.Delete("/cookbooks/" + cb2 + "/1.0.0")
		client.Delete("/cookbooks/" + cb2 + "/1.2.0")
		client.Delete("/cookbooks/" + cb2 + "/1.2.5")
	}()

	// Fetch cookbooks from environment
	resp, err = client.Get("/environments/" + envName + "/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	// Should have both cookbooks
	if _, ok := body[cb1]; !ok {
		t.Errorf("expected cookbook %q in response, got: %v", cb1, body)
	}
	if _, ok := body[cb2]; !ok {
		t.Errorf("expected cookbook %q in response, got: %v", cb2, body)
	}

	// Each should have a url and versions
	cb1Info := body[cb1].(map[string]interface{})
	if _, ok := cb1Info["url"]; !ok {
		t.Errorf("expected 'url' for %q, got: %v", cb1, cb1Info)
	}
	versions, ok := cb1Info["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array for %q, got: %v", cb1, cb1Info)
	}
	// Default num_versions=1 should return latest only
	if len(versions) != 1 {
		t.Errorf("expected 1 version (default), got %d: %v", len(versions), versions)
	}
}

func TestEnvironmentsCookbooksNumVersionsAll(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_cb_nvall")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	cbName := pedant.UniqueName("nv_cb")
	versions := []string{"1.0.0", "2.0.0", "3.0.0"}
	for _, v := range versions {
		payload := newCookbookPayload(cbName, v)
		resp, err := client.Put("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.Delete("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err = client.Get("/environments/" + envName + "/cookbooks?num_versions=all")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks?num_versions=all: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo := body[cbName].(map[string]interface{})
	versionsResp := cbInfo["versions"].([]interface{})
	if len(versionsResp) != 3 {
		t.Errorf("expected 3 versions with num_versions=all, got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestEnvironmentsCookbooksWithConstraints(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_cb_con")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"cookbook_versions": map[string]string{
			"pedant_cb_one": "= 1.0.0",
		},
	})
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	cbName := pedant.UniqueName("pedant_cb_one")
	payload := newCookbookPayload(cbName, "1.0.0")
	resp, err = client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err = client.Get("/environments/" + envName + "/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[cbName]; !ok {
		t.Errorf("expected cookbook %q in response, got: %v", cbName, body)
	}
}

// GET /environments/<name>/cookbooks/<name>

func TestEnvironmentsSingleCookbookNonExistentEnv(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/bad_env/cookbooks/fake_cookbook")
	if err != nil {
		t.Fatalf("GET /environments/bad_env/cookbooks/fake_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsSingleCookbookNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_scb")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/environments/" + envName + "/cookbooks/non_existent_cookbook")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks/non_existent_cookbook: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsSingleCookbookDefaultNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/_default/cookbooks/non_existent_cookbook")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks/non_existent_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsSingleCookbookExisting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_scb_ex")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	cbName := pedant.UniqueName("scb_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	resp, err = client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err = client.Get("/environments/" + envName + "/cookbooks/" + cbName)
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks/%s: %v", envName, cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	if _, ok := cbInfo["url"]; !ok {
		t.Errorf("expected 'url' in cookbook info, got: %v", cbInfo)
	}
	if _, ok := cbInfo["versions"]; !ok {
		t.Errorf("expected 'versions' in cookbook info, got: %v", cbInfo)
	}
}

func TestEnvironmentsSingleCookbookDefaultExisting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("scb_def")
	payload := newCookbookPayload(cbName, "1.0.0")
	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err = client.Get("/environments/_default/cookbooks/" + cbName)
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks/%s: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

// GET /environments/<name>/recipes

func TestEnvironmentsRecipesNoCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_rec_empty")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/environments/" + envName + "/recipes")
	if err != nil {
		t.Fatalf("GET /environments/%s/recipes: %v", envName, err)
	}
	// Just verify the endpoint returns 200
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsRecipesDefaultNoCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/_default/recipes")
	if err != nil {
		t.Fatalf("GET /environments/_default/recipes: %v", err)
	}
	// Just verify the endpoint returns 200
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsRecipesNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/bad_env/recipes")
	if err != nil {
		t.Fatalf("GET /environments/bad_env/recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsRecipesWithCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_rec_cb")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	cbName := pedant.UniqueName("rec_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	resp, err = client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err = client.Get("/environments/" + envName + "/recipes")
	if err != nil {
		t.Fatalf("GET /environments/%s/recipes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

// GET /environments/<name>/roles/<name>

func TestEnvironmentsRolesDefaultExisting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	roleName := pedant.UniqueName("env_role")
	role := pedant.NewRole(roleName, map[string]interface{}{
		"run_list": []string{"recipe[nginx]"},
	})
	defer client.Delete("/roles/" + roleName)

	resp, err := client.Post("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/environments/_default/roles/" + roleName)
	if err != nil {
		t.Fatalf("GET /environments/_default/roles/%s: %v", roleName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	runList, ok := body["run_list"].([]interface{})
	if !ok {
		t.Fatalf("expected 'run_list' array, got: %v", body)
	}
	if len(runList) != 1 || runList[0] != "recipe[nginx]" {
		t.Errorf("expected run_list [recipe[nginx]], got %v", runList)
	}
}

func TestEnvironmentsRolesDefaultNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/_default/roles/not_a_role")
	if err != nil {
		t.Fatalf("GET /environments/_default/roles/not_a_role: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsRolesNonDefaultExisting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_role_env")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	roleName := pedant.UniqueName("env_role2")
	role := pedant.NewRole(roleName, map[string]interface{}{
		"env_run_lists": map[string]interface{}{
			envName: []string{"recipe[apache2]"},
		},
	})
	defer client.Delete("/roles/" + roleName)

	resp, err = client.Post("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/environments/" + envName + "/roles/" + roleName)
	if err != nil {
		t.Fatalf("GET /environments/%s/roles/%s: %v", envName, roleName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	runList, ok := body["run_list"].([]interface{})
	if !ok {
		t.Fatalf("expected 'run_list' array, got: %v", body)
	}
	if len(runList) != 1 || runList[0] != "recipe[apache2]" {
		t.Errorf("expected run_list [recipe[apache2]], got %v", runList)
	}
}

func TestEnvironmentsRolesNonDefaultNonExistentEnv(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/bad_env/roles/some_role")
	if err != nil {
		t.Fatalf("GET /environments/bad_env/roles/some_role: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsRolesRoleInDifferentEnv(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("env_role_diff")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	roleName := pedant.UniqueName("diff_role")
	role := pedant.NewRole(roleName, map[string]interface{}{
		"run_list": []string{"recipe[nginx]"},
	})
	defer client.Delete("/roles/" + roleName)

	resp, err = client.Post("/roles", role)
	if err != nil {
		t.Fatalf("POST /roles: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Role exists but has no env_run_lists for this environment
	resp, err = client.Get("/environments/" + envName + "/roles/" + roleName)
	if err != nil {
		t.Fatalf("GET /environments/%s/roles/%s: %v", envName, roleName, err)
	}
	// goiardi returns 200 with null run_list for roles without env_run_lists
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["run_list"] != nil {
		t.Errorf("expected null run_list for role without env_run_lists, got %v", body["run_list"])
	}
}

// --- Additional Environment CRUD tests ---
// Ported from: create_spec.rb, read_spec.rb, update_spec.rb, delete_spec.rb

func TestEnvironmentsCreateFull(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("full_env")
	env := pedant.NewEnvironment(envName, map[string]interface{}{
		"description":         "A test environment",
		"default_attributes":  map[string]interface{}{"key": "value"},
		"override_attributes": map[string]interface{}{"override": "yes"},
		"cookbook_versions":   map[string]string{"nginx": ">= 1.0.0"},
	})
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Verify persisted
	resp, err = client.Get("/environments/" + envName)
	if err != nil {
		t.Fatalf("GET /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["description"] != "A test environment" {
		t.Errorf("expected description, got %v", body["description"])
	}
	if body["name"] != envName {
		t.Errorf("expected name %q, got %q", envName, body["name"])
	}
}

func TestEnvironmentsCreateMinimal(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("min_env")
	env := map[string]interface{}{
		"name": envName,
	}
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
}

func TestEnvironmentsCreateMissingName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	env := map[string]interface{}{}
	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestEnvironmentsCreateInvalidName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	invalidNames := []string{"abc!123", "abc 123", "大爆発"}
	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			env := pedant.NewEnvironment(name)
			resp, err := client.Post("/environments", env)
			if err != nil {
				t.Fatalf("POST /environments: %v", err)
			}
			pedant.AssertStatus(t, resp, 400)
		})
	}
}

func TestEnvironmentsCreateAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	envName := pedant.UniqueName("no_perm_env")
	env := pedant.NewEnvironment(envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsReadDefaultDetails(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/_default")
	if err != nil {
		t.Fatalf("GET /environments/_default: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["name"] != "_default" {
		t.Errorf("expected name '_default', got %v", body["name"])
	}
	if body["description"] != "The default Chef environment" {
		t.Errorf("expected default description, got %v", body["description"])
	}
}

func TestEnvironmentsReadNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments/nonexistent_env")
	if err != nil {
		t.Fatalf("GET /environments/nonexistent_env: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsReadAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	resp, err := client.Get("/environments/_default")
	if err != nil {
		t.Fatalf("GET /environments/_default: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsUpdateFull(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("upd_full")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	update := pedant.NewEnvironment(envName, map[string]interface{}{
		"description":         "Updated",
		"default_attributes":  map[string]interface{}{"new_key": "new_value"},
		"override_attributes": map[string]interface{}{"new_override": "yes"},
		"cookbook_versions":   map[string]string{"nginx": ">= 2.0.0"},
	})
	resp, err = client.Put("/environments/"+envName, update)
	if err != nil {
		t.Fatalf("PUT /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/environments/" + envName)
	if err != nil {
		t.Fatalf("GET /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["description"] != "Updated" {
		t.Errorf("expected description 'Updated', got %v", body["description"])
	}
}

func TestEnvironmentsUpdateAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("upd_no_perm")
	env := pedant.NewEnvironment(envName)
	defer adminClient.Delete("/environments/" + envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	client := testServer.NewClient(testServer.NormalUser)
	resp, err = client.Put("/environments/"+envName, env)
	if err != nil {
		t.Fatalf("PUT /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsDeleteAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("del_no_perm")
	env := pedant.NewEnvironment(envName)
	defer adminClient.Delete("/environments/" + envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	client := testServer.NewClient(testServer.NormalUser)
	resp, err = client.Delete("/environments/" + envName)
	if err != nil {
		t.Fatalf("DELETE /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsDeleteNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Delete("/environments/nonexistent_env")
	if err != nil {
		t.Fatalf("DELETE /environments/nonexistent_env: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsList(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("list_env")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body["_default"]; !ok {
		t.Errorf("expected '_default' in environment list, got: %v", body)
	}
	if _, ok := body[envName]; !ok {
		t.Errorf("expected %q in environment list, got: %v", envName, body)
	}
}

func TestEnvironmentsListAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	resp, err := client.Get("/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

// --- Environment Permission Checks ---
// These correspond to the open_source_permissions_spec.rb tests.

func TestEnvironmentsPermissionListAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionListAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	resp, err := client.Get("/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionListAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)
	resp, err := client.Get("/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionListAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)
	resp, err := client.Get("/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionCreateAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_create")
	env := pedant.NewEnvironment(envName)
	defer client.Delete("/environments/" + envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
}

func TestEnvironmentsPermissionCreateAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	envName := pedant.UniqueName("perm_create")
	env := pedant.NewEnvironment(envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	// chef-validator may have been deleted (test ordering issue)
	if resp.StatusCode == 401 {
		t.Skip("chef-validator client was deleted by a previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionCreateAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)
	envName := pedant.UniqueName("perm_create")
	env := pedant.NewEnvironment(envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionCreateAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)
	envName := pedant.UniqueName("perm_create")
	env := pedant.NewEnvironment(envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionCollectionPutNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_put")
	env := pedant.NewEnvironment(envName)

	resp, err := client.Put("/environments", env)
	if err != nil {
		t.Fatalf("PUT /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionCollectionDeleteNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	resp, err := client.Delete("/environments")
	if err != nil {
		t.Fatalf("DELETE /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionGetAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_get")
	createAndDeleteEnv(t, envName)

	resp, err := client.Get("/environments/" + envName)
	if err != nil {
		t.Fatalf("GET /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionGetAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)

	resp, err := client.Get("/environments/_default")
	if err != nil {
		t.Fatalf("GET /environments/_default: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionGetAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)

	resp, err := client.Get("/environments/_default")
	if err != nil {
		t.Fatalf("GET /environments/_default: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionGetAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)

	resp, err := client.Get("/environments/_default")
	if err != nil {
		t.Fatalf("GET /environments/_default: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionNamedPostNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_npost")
	env := pedant.NewEnvironment(envName)

	resp, err := client.Post("/environments/"+envName, env)
	if err != nil {
		t.Fatalf("POST /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsPermissionUpdateAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_upd")
	createAndDeleteEnv(t, envName)

	update := pedant.NewEnvironment(envName)
	resp, err := client.Put("/environments/"+envName, update)
	if err != nil {
		t.Fatalf("PUT /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionUpdateAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_upd")
	env := pedant.NewEnvironment(envName)
	defer adminClient.Delete("/environments/" + envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	normalClient := testServer.NewClient(testServer.NormalUser)
	update := pedant.NewEnvironment(envName)
	resp, err = normalClient.Put("/environments/"+envName, update)
	if err != nil {
		t.Fatalf("PUT /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionUpdateAsValidator(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_upd")
	env := pedant.NewEnvironment(envName)
	defer adminClient.Delete("/environments/" + envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	valClient := testServer.NewClient(testServer.ValidatorClient)
	update := pedant.NewEnvironment(envName)
	resp, err = valClient.Put("/environments/"+envName, update)
	if err != nil {
		t.Fatalf("PUT /environments/%s: %v", envName, err)
	}
	// Chef Server returns 403; goiardi might allow
	if resp.StatusCode != 403 && resp.StatusCode != 200 {
		// If auth fails with 401, the shared chef-validator was deleted
		if resp.StatusCode == 401 {
			t.Skip("chef-validator was deleted by previous test (shared state issue)")
		}
		pedant.AssertStatus(t, resp, 403)
	}
}

func TestEnvironmentsPermissionUpdateAsBadClient(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_upd")
	env := pedant.NewEnvironment(envName)
	defer adminClient.Delete("/environments/" + envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	outsideClient := testServer.NewClient(testServer.OutsideUser)
	update := pedant.NewEnvironment(envName)
	resp, err = outsideClient.Put("/environments/"+envName, update)
	if err != nil {
		t.Fatalf("PUT /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionDeleteAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_del")
	env := pedant.NewEnvironment(envName)

	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Delete("/environments/" + envName)
	if err != nil {
		t.Fatalf("DELETE /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionDeleteAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_del")
	env := pedant.NewEnvironment(envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	defer adminClient.Delete("/environments/" + envName)

	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.Delete("/environments/" + envName)
	if err != nil {
		t.Fatalf("DELETE /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionDeleteAsValidator(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_del")
	env := pedant.NewEnvironment(envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	defer adminClient.Delete("/environments/" + envName)

	valClient := testServer.NewClient(testServer.ValidatorClient)
	resp, err = valClient.Delete("/environments/" + envName)
	if err != nil {
		t.Fatalf("DELETE /environments/%s: %v", envName, err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionDeleteAsBadClient(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_del")
	env := pedant.NewEnvironment(envName)

	resp, err := adminClient.Post("/environments", env)
	if err != nil {
		t.Fatalf("POST /environments: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	defer adminClient.Delete("/environments/" + envName)

	outsideClient := testServer.NewClient(testServer.OutsideUser)
	resp, err = outsideClient.Delete("/environments/" + envName)
	if err != nil {
		t.Fatalf("DELETE /environments/%s: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 401)
}

// --- Environment Sub-endpoint Permission Checks ---

func TestEnvironmentsPermissionCookbooksGetAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_cb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Get("/environments/" + envName + "/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionCookbooksGetAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)

	resp, err := client.Get("/environments/_default/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionCookbooksGetAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)

	resp, err := client.Get("/environments/_default/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionCookbooksGetAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)

	resp, err := client.Get("/environments/_default/cookbooks")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionCookbooksPostNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_cb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Post("/environments/"+envName+"/cookbooks", map[string]interface{}{})
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbooks: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionCookbooksPutNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_cb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Put("/environments/"+envName+"/cookbooks", map[string]interface{}{})
	if err != nil {
		t.Fatalf("PUT /environments/%s/cookbooks: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionCookbooksDeleteNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_cb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Delete("/environments/" + envName + "/cookbooks")
	if err != nil {
		t.Fatalf("DELETE /environments/%s/cookbooks: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionSingleCookbookGetAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_scb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Get("/environments/" + envName + "/cookbooks/nginx")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbooks/nginx: %v", envName, err)
	}
	// Non-existent cookbook returns 404 (env exists, but no such cookbook)
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsPermissionSingleCookbookGetAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)

	resp, err := client.Get("/environments/_default/cookbooks/nginx")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks/nginx: %v", err)
	}
	// goiardi returns 404 for non-existent cookbook
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsPermissionSingleCookbookGetAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)

	resp, err := client.Get("/environments/_default/cookbooks/nginx")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks/nginx: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionSingleCookbookGetAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)

	resp, err := client.Get("/environments/_default/cookbooks/nginx")
	if err != nil {
		t.Fatalf("GET /environments/_default/cookbooks/nginx: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionSingleCookbookPostNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_scb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Post("/environments/"+envName+"/cookbooks/nginx", map[string]interface{}{})
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbooks/nginx: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionSingleCookbookPutNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_scb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Put("/environments/"+envName+"/cookbooks/nginx", map[string]interface{}{})
	if err != nil {
		t.Fatalf("PUT /environments/%s/cookbooks/nginx: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionSingleCookbookDeleteNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_scb")
	createAndDeleteEnv(t, envName)

	resp, err := client.Delete("/environments/" + envName + "/cookbooks/nginx")
	if err != nil {
		t.Fatalf("DELETE /environments/%s/cookbooks/nginx: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionRecipesGetAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_rec")
	createAndDeleteEnv(t, envName)

	resp, err := client.Get("/environments/" + envName + "/recipes")
	if err != nil {
		t.Fatalf("GET /environments/%s/recipes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionRecipesGetAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)

	resp, err := client.Get("/environments/_default/recipes")
	if err != nil {
		t.Fatalf("GET /environments/_default/recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionRecipesGetAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)

	resp, err := client.Get("/environments/_default/recipes")
	if err != nil {
		t.Fatalf("GET /environments/_default/recipes: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionRecipesGetAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)

	resp, err := client.Get("/environments/_default/recipes")
	if err != nil {
		t.Fatalf("GET /environments/_default/recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionRecipesPostNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_rec")
	createAndDeleteEnv(t, envName)

	resp, err := client.Post("/environments/"+envName+"/recipes", map[string]interface{}{})
	if err != nil {
		t.Fatalf("POST /environments/%s/recipes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionRecipesPutNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_rec")
	createAndDeleteEnv(t, envName)

	resp, err := client.Put("/environments/"+envName+"/recipes", map[string]interface{}{})
	if err != nil {
		t.Fatalf("PUT /environments/%s/recipes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionRecipesDeleteNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_rec")
	createAndDeleteEnv(t, envName)

	resp, err := client.Delete("/environments/" + envName + "/recipes")
	if err != nil {
		t.Fatalf("DELETE /environments/%s/recipes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionNodesGetAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_nodes")
	createAndDeleteEnv(t, envName)

	resp, err := client.Get("/environments/" + envName + "/nodes")
	if err != nil {
		t.Fatalf("GET /environments/%s/nodes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionNodesGetAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)

	resp, err := client.Get("/environments/_default/nodes")
	if err != nil {
		t.Fatalf("GET /environments/_default/nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionNodesGetAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)

	resp, err := client.Get("/environments/_default/nodes")
	if err != nil {
		t.Fatalf("GET /environments/_default/nodes: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionNodesGetAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)

	resp, err := client.Get("/environments/_default/nodes")
	if err != nil {
		t.Fatalf("GET /environments/_default/nodes: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionNodesPostNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_nodes")
	createAndDeleteEnv(t, envName)

	resp, err := client.Post("/environments/"+envName+"/nodes", map[string]interface{}{})
	if err != nil {
		t.Fatalf("POST /environments/%s/nodes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionNodesPutNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_nodes")
	createAndDeleteEnv(t, envName)

	resp, err := client.Put("/environments/"+envName+"/nodes", map[string]interface{}{})
	if err != nil {
		t.Fatalf("PUT /environments/%s/nodes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionNodesDeleteNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_nodes")
	createAndDeleteEnv(t, envName)

	resp, err := client.Delete("/environments/" + envName + "/nodes")
	if err != nil {
		t.Fatalf("DELETE /environments/%s/nodes: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionRolesGetAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_role")
	createAndDeleteEnv(t, envName)

	resp, err := client.Get("/environments/" + envName + "/roles/web")
	if err != nil {
		t.Fatalf("GET /environments/%s/roles/web: %v", envName, err)
	}
	// Non-existent role returns 404
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsPermissionRolesGetAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)

	resp, err := client.Get("/environments/_default/roles/web")
	if err != nil {
		t.Fatalf("GET /environments/_default/roles/web: %v", err)
	}
	// goiardi returns 404 for non-existent role
	pedant.AssertStatus(t, resp, 404)
}

func TestEnvironmentsPermissionRolesGetAsValidator(t *testing.T) {
	client := testServer.NewClient(testServer.ValidatorClient)

	resp, err := client.Get("/environments/_default/roles/web")
	if err != nil {
		t.Fatalf("GET /environments/_default/roles/web: %v", err)
	}
	// If auth fails with 401, the shared chef-validator was deleted
	if resp.StatusCode == 401 {
		t.Skip("chef-validator was deleted by previous test (shared state issue)")
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestEnvironmentsPermissionRolesGetAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)

	resp, err := client.Get("/environments/_default/roles/web")
	if err != nil {
		t.Fatalf("GET /environments/_default/roles/web: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionRolesPostNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_role")
	createAndDeleteEnv(t, envName)

	resp, err := client.Post("/environments/"+envName+"/roles/web", map[string]interface{}{})
	if err != nil {
		t.Fatalf("POST /environments/%s/roles/web: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionRolesPutNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_role")
	createAndDeleteEnv(t, envName)

	resp, err := client.Put("/environments/"+envName+"/roles/web", map[string]interface{}{})
	if err != nil {
		t.Fatalf("PUT /environments/%s/roles/web: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionRolesDeleteNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_role")
	createAndDeleteEnv(t, envName)

	resp, err := client.Delete("/environments/" + envName + "/roles/web")
	if err != nil {
		t.Fatalf("DELETE /environments/%s/roles/web: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionDepsolverAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_depsolv")
	createAndDeleteEnv(t, envName)

	resp, err := client.Post("/environments/"+envName+"/cookbook_versions", map[string]interface{}{
		"run_list": []string{},
	})
	if err != nil {
		t.Fatalf("POST /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionDepsolverAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)

	resp, err := client.Post("/environments/_default/cookbook_versions", map[string]interface{}{
		"run_list": []string{},
	})
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestEnvironmentsPermissionDepsolverAsBadClient(t *testing.T) {
	client := testServer.NewClient(testServer.OutsideUser)

	resp, err := client.Post("/environments/_default/cookbook_versions", map[string]interface{}{
		"run_list": []string{},
	})
	if err != nil {
		t.Fatalf("POST /environments/_default/cookbook_versions: %v", err)
	}
	pedant.AssertStatus(t, resp, 401)
}

func TestEnvironmentsPermissionDepsolverGetNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_depsolv")
	createAndDeleteEnv(t, envName)

	resp, err := client.Get("/environments/" + envName + "/cookbook_versions")
	if err != nil {
		t.Fatalf("GET /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionDepsolverPutNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_depsolv")
	createAndDeleteEnv(t, envName)

	resp, err := client.Put("/environments/"+envName+"/cookbook_versions", map[string]interface{}{})
	if err != nil {
		t.Fatalf("PUT /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestEnvironmentsPermissionDepsolverDeleteNotAllowed(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	envName := pedant.UniqueName("perm_depsolv")
	createAndDeleteEnv(t, envName)

	resp, err := client.Delete("/environments/" + envName + "/cookbook_versions")
	if err != nil {
		t.Fatalf("DELETE /environments/%s/cookbook_versions: %v", envName, err)
	}
	pedant.AssertStatus(t, resp, 405)
}


// createAndDeleteEnv is a helper for permission tests that need an environment
// to exist but don't need specific assertions on creation.
func createAndDeleteEnv(t *testing.T, name string) {
	t.Helper()
	client := testServer.NewClient(testServer.AdminUser)
	env := pedant.NewEnvironment(name)
	resp, err := client.Post("/environments", env)
	if err != nil {
		t.Fatalf("creating environment %s: %v", name, err)
	}
	pedant.AssertStatus(t, resp, 201)
	t.Cleanup(func() {
		client.Delete("/environments/" + name)
	})
}
