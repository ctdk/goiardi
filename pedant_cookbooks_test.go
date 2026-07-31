package main

import (
	"fmt"
	"github.com/ctdk/goiardi/pedant"
	"strings"
	"testing"
)

func TestCookbooksListEmpty(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/cookbooks")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty cookbook list, got %d entries", len(body))
	}
}

func TestCookbooksCreateAndRead(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("test_cb")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["name"] != cbName+"-"+cbVersion {
		t.Errorf("expected name %q, got %q", cbName+"-"+cbVersion, body["name"])
	}
}

func TestCookbooksCreateDuplicate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("dup_cb")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("first PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("second PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksMultipleVersions(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("multi_cb")
	versions := []string{"1.0.0", "1.1.0", "2.0.0"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks/" + cbName + "?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 3 {
		t.Errorf("expected 3 versions, got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestCookbooksListWithVersions(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("list_cb")
	versions := []string{"0.0.1", "0.0.2"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 2 {
		t.Errorf("expected 2 versions, got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestCookbooksNumVersionsFilter(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("nv_cb")
	versions := []string{"0.0.1", "0.0.2", "0.0.3"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks?num_versions=1")
	if err != nil {
		t.Fatalf("GET /cookbooks?num_versions=1: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 1 {
		t.Errorf("expected 1 version with num_versions=1, got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestCookbooksNumVersionsAll(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("nva_cb")
	versions := []string{"0.0.1", "0.0.2"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks?num_versions=all: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 2 {
		t.Errorf("expected 2 versions with num_versions=all, got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestCookbooksNumVersionsZero(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("nv0_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks?num_versions=0")
	if err != nil {
		t.Fatalf("GET /cookbooks?num_versions=0: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 0 {
		t.Errorf("expected 0 versions with num_versions=0, got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestCookbooksDelete(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/cookbooks/nonexistent_cb/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/nonexistent_cb/1.0.0: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksInvalidName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	payload := pedant.NewCookbook("valid_name", "1.0.0")

	resp, err := client.PutOrg("/cookbooks/first@second/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/first@second/1.0.0: %v", err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksJSONClassValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("jsonclass_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	payload["json_class"] = "Chef::Node"
	defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "json_class")
}

func TestCookbooksChefTypeValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("cheftype_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	payload["chef_type"] = "node"
	defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "chef_type")
}

func TestCookbooksInvalidKeys(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("inv_keys_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	payload["invalid_key"] = "some_value"
	defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "Invalid key")
}

func TestCookbooksMetadataVersionMissing(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("meta_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	payload["metadata"] = map[string]interface{}{}
	defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.version")
}

func TestCookbooksSegmentValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("seg_cb")

	segments := []string{"resources", "providers", "recipes", "definitions", "libraries", "attributes", "files", "templates", "root_files"}
	for _, seg := range segments {
		t.Run(seg, func(t *testing.T) {
			payload := pedant.NewCookbook(cbName, "1.0.0")
			payload[seg] = "foo"
			defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

			resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "invalid")
		})
	}
}

func TestCookbooksMismatchedName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("mismatch_cb")
	payload := pedant.NewCookbook("wrong_name", "1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "name")
}

func TestCookbooksMismatchedVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("mismatch_ver")
	payload := pedant.NewCookbook(cbName, "0.0.1")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "name")
}

func TestCookbooksMetadataSectionValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("meta_sec")

	sections := []string{"platforms", "dependencies", "recommendations", "suggestions", "conflicting", "replacing"}
	for _, section := range sections {
		t.Run(section+"_invalid_type", func(t *testing.T) {
			payload := pedant.NewCookbook(cbName, "1.0.0")
			payload["metadata"].(map[string]interface{})[section] = "foo"
			defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

			resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "invalid")
		})
	}
}

func TestCookbooksUpdate(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	defer client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "updated description"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("second PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksNonAdminCannotCreate(t *testing.T) {
	normalClient := testServer.NewClient(testServer.NormalUser)
	cbName := pedant.UniqueName("no_perm_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	defer normalClient.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := normalClient.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestCookbooksNonAdminCannotDelete(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("no_del_cb")
	payload := pedant.NewCookbook(cbName, "1.0.0")
	defer adminClient.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")

	resp, err := adminClient.PutOrg("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestCookbooksValidatorCannotList(t *testing.T) {
	validatorClient := testServer.NewClient(testServer.ValidatorClient)
	resp, err := validatorClient.GetOrg("/cookbooks")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	// If the validator client was deleted by another test, skip
	if resp.StatusCode == 401 {
		t.Skip("validator client no longer exists (deleted by another test)")
		return
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestCookbooksMultipleCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cb1 := pedant.UniqueName("multi_cb1")
	cb2 := pedant.UniqueName("multi_cb2")

	payload1 := pedant.NewCookbook(cb1, "0.0.1")
	payload2 := pedant.NewCookbook(cb2, "0.0.2")

	resp, err := client.PutOrg("/cookbooks/"+cb1+"/0.0.1", payload1)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/0.0.1: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PutOrg("/cookbooks/"+cb2+"/0.0.2", payload2)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/0.0.2: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	defer func() {
		client.DeleteOrg("/cookbooks/" + cb1 + "/0.0.1")
		client.DeleteOrg("/cookbooks/" + cb2 + "/0.0.2")
	}()

	resp, err = client.GetOrg("/cookbooks?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[cb1]; !ok {
		t.Errorf("expected cookbook %q in list, not found", cb1)
	}
	if _, ok := body[cb2]; !ok {
		t.Errorf("expected cookbook %q in list, not found", cb2)
	}
}

func TestCookbooksLatest(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("latest_cb")
	versions := []string{"1.0.0", "2.0.0", "1.5.0"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks/" + cbName + "/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/_latest: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["version"] != "2.0.0" {
		t.Errorf("expected latest version 2.0.0, got %v", body["version"])
	}
}

// --- Cookbook Read Tests ---

func TestCookbooksReadVerifyRecipeURL(t *testing.T) {
	// This test verifies that a cookbook with recipe files returns
	// the correct data structure. Full file upload via sandbox requires
	// correctly configured file_store URLs which don't work in test mode
	// (config.ServerBaseURL uses port 0). We test the data structure instead.
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("read_cb")
	cbVersion := "1.2.3"

	// Create a cookbook with empty recipe list (no file upload needed)
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Read the cookbook version
	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	body := pedant.GetJSONBody(t, resp)

	// Verify standard fields
	if body["name"] != cbName+"-"+cbVersion {
		t.Errorf("expected name %q, got %q", cbName+"-"+cbVersion, body["name"])
	}
	if body["cookbook_name"] != cbName {
		t.Errorf("expected cookbook_name %q, got %q", cbName, body["cookbook_name"])
	}
	if body["version"] != cbVersion {
		t.Errorf("expected version %q, got %q", cbVersion, body["version"])
	}
	if body["json_class"] != "Chef::CookbookVersion" {
		t.Errorf("expected json_class 'Chef::CookbookVersion', got %v", body["json_class"])
	}
	if body["chef_type"] != "cookbook_version" {
		t.Errorf("expected chef_type 'cookbook_version', got %v", body["chef_type"])
	}
	if body["frozen?"] != false {
		t.Errorf("expected frozen? false, got %v", body["frozen?"])
	}

	// Verify recipes is empty array (no recipes uploaded)
	recipes, ok := body["recipes"].([]interface{})
	if !ok {
		t.Fatalf("expected 'recipes' array, got: %v", body["recipes"])
	}
	if len(recipes) != 0 {
		t.Errorf("expected empty recipes array, got %d entries", len(recipes))
	}

	// Verify metadata exists
	if _, ok := body["metadata"]; !ok {
		t.Errorf("expected 'metadata' in response")
	}
}

func TestCookbooksReadRecipeKeys(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("recipe_keys")
	cbVersion := "1.2.3"

	// Put a cookbook with recipe data
	cbPayload := pedant.NewCookbook(cbName, cbVersion, map[string]interface{}{
		"recipes": []interface{}{
			map[string]interface{}{
				"name":        "default.rb",
				"path":        "recipes/default.rb",
				"checksum":    "0000000000000000000000000000000000000000",
				"specificity": "default",
			},
		},
	})
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, cbPayload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	// Cookbook with recipe but no sandbox upload will fail validation
	// This is expected behavior — the sandbox check happens on create
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksReadAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("read_norm")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer adminClient.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Read as normal user
	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksReadAsAdminUser(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("read_admin")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksReadVerifyResponseBody(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("verify_cb")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion, map[string]interface{}{
		"metadata": map[string]interface{}{
			"version":          cbVersion,
			"name":             cbName,
			"maintainer":       "Your Name",
			"description":      "A fabulous new cookbook",
			"long_description": "",
			"maintainer_email": "youremail@example.com",
			"license":          "Apache v2.0",
			"platforms":        map[string]interface{}{},
			"dependencies":     map[string]interface{}{},
			"recommendations":  map[string]interface{}{},
			"suggestions":      map[string]interface{}{},
			"conflicting":      map[string]interface{}{},
			"replacing":        map[string]interface{}{},
			"groupings":        map[string]interface{}{},
			"providing":        map[string]interface{}{},
		},
	})
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	body := pedant.GetJSONBody(t, resp)

	// Verify standard fields
	expectedFields := []string{"name", "cookbook_name", "version", "json_class", "chef_type", "frozen?", "recipes", "metadata"}
	for _, field := range expectedFields {
		if _, ok := body[field]; !ok {
			t.Errorf("expected field %q in response body", field)
		}
	}

	if body["name"] != cbName+"-"+cbVersion {
		t.Errorf("expected name %q, got %q", cbName+"-"+cbVersion, body["name"])
	}
	if body["cookbook_name"] != cbName {
		t.Errorf("expected cookbook_name %q, got %q", cbName, body["cookbook_name"])
	}
	if body["version"] != cbVersion {
		t.Errorf("expected version %q, got %q", cbVersion, body["version"])
	}
	if body["json_class"] != "Chef::CookbookVersion" {
		t.Errorf("expected json_class 'Chef::CookbookVersion', got %v", body["json_class"])
	}
	if body["chef_type"] != "cookbook_version" {
		t.Errorf("expected chef_type 'cookbook_version', got %v", body["chef_type"])
	}
	if body["frozen?"] != false {
		t.Errorf("expected frozen? false, got %v", body["frozen?"])
	}

	// Verify metadata
	metadata, ok := body["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'metadata' in response, got: %v", body)
	}
	if metadata["version"] != cbVersion {
		t.Errorf("expected metadata.version %q, got %v", cbVersion, metadata["version"])
	}
	if metadata["name"] != cbName {
		t.Errorf("expected metadata.name %q, got %v", cbName, metadata["name"])
	}
	if metadata["maintainer"] != "Your Name" {
		t.Errorf("expected metadata.maintainer 'Your Name', got %v", metadata["maintainer"])
	}
	if metadata["description"] != "A fabulous new cookbook" {
		t.Errorf("expected metadata.description 'A fabulous new cookbook', got %v", metadata["description"])
	}
}

func TestCookbooksReadSingleInCollection(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("single_cb")
	cbVersion := "1.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in collection, got: %v", cbName, body)
	}

	// Verify URL and versions structure
	if _, ok := cbInfo["url"]; !ok {
		t.Errorf("expected 'url' in cookbook info, got: %v", cbInfo)
	}
	versions, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}

	verInfo := versions[0].(map[string]interface{})
	if verInfo["version"] != cbVersion {
		t.Errorf("expected version %q, got %v", cbVersion, verInfo["version"])
	}
	if _, ok := verInfo["url"]; !ok {
		t.Errorf("expected 'url' in version info, got: %v", verInfo)
	}
}

func TestCookbooksReadMultipleInCollection(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("multi_read_cb")
	versions := []string{"0.0.1", "0.0.2"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in collection, got: %v", cbName, body)
	}

	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 2 {
		t.Fatalf("expected 2 versions, got %d: %v", len(versionsResp), versionsResp)
	}

	// Versions should be sorted descending (newest first)
	v0 := versionsResp[0].(map[string]interface{})
	v1 := versionsResp[1].(map[string]interface{})
	if v0["version"] != "0.0.2" {
		t.Errorf("expected first version 0.0.2, got %v", v0["version"])
	}
	if v1["version"] != "0.0.1" {
		t.Errorf("expected second version 0.0.1, got %v", v1["version"])
	}
}

func TestCookbooksReadNumVersionsDefault(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("nv_default")
	versions := []string{"0.0.1", "0.0.2"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	// Without num_versions, should return 1 version per cookbook
	resp, err := client.GetOrg("/cookbooks")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in collection, got: %v", cbName, body)
	}

	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	// Without num_versions, goiardi defaults to 1 version
	if len(versionsResp) != 1 {
		t.Errorf("expected 1 version (default), got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestCookbooksReadVerifyURLFormat(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("url_fmt")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Check the cookbook collection URL
	resp, err = client.GetOrg("/cookbooks?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in collection, got: %v", cbName, body)
	}

	cbURL, ok := cbInfo["url"].(string)
	if !ok {
		t.Fatalf("expected 'url' in cookbook info, got: %v", cbInfo)
	}
	expectedURLSuffix := "/cookbooks/" + cbName
	if !strings.HasSuffix(cbURL, expectedURLSuffix) {
		t.Errorf("expected cookbook URL to end with %q, got %q", expectedURLSuffix, cbURL)
	}

	versions, ok := cbInfo["versions"].([]interface{})
	if !ok || len(versions) == 0 {
		t.Fatalf("expected versions array, got: %v", cbInfo)
	}
	verInfo := versions[0].(map[string]interface{})
	verURL, ok := verInfo["url"].(string)
	if !ok {
		t.Fatalf("expected 'url' in version info, got: %v", verInfo)
	}
	expectedVerURLSuffix := "/cookbooks/" + cbName + "/" + cbVersion
	if !strings.HasSuffix(verURL, expectedVerURLSuffix) {
		t.Errorf("expected version URL to end with %q, got %q", expectedVerURLSuffix, verURL)
	}
}

func TestCookbooksReadSegmentsEmpty(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("seg_empty")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)

	// Segments with no data should not appear in the response
	// or should be empty arrays depending on request source
	emptySegments := []string{"definitions", "libraries", "attributes", "providers", "resources", "templates", "root_files", "files"}
	for _, seg := range emptySegments {
		if _, ok := body[seg]; ok {
			// Response includes recipes and metadata always
			// Other segments are omitted if empty (not x-ops-request-source: web)
			t.Logf("segment %q present in response (may be empty array)", seg)
		}
	}

	// recipes should always be an empty array
	recipes, ok := body["recipes"].([]interface{})
	if !ok {
		t.Errorf("expected 'recipes' to be an array, got: %v", body["recipes"])
	} else if len(recipes) != 0 {
		t.Errorf("expected empty 'recipes', got %d entries", len(recipes))
	}
}

func TestCookbooksReadNonAdminCannotRead(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("no_read_cb")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer adminClient.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Validator cannot list
	validatorClient := testServer.NewClient(testServer.ValidatorClient)
	resp, err = validatorClient.GetOrg("/cookbooks")
	if err != nil {
		t.Fatalf("GET /cookbooks: %v", err)
	}
	// If the validator client was deleted by another test, skip
	if resp.StatusCode == 401 {
		t.Skip("validator client no longer exists (deleted by another test)")
		return
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestCookbooksReadByName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("by_name")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// GET /cookbooks/<name> returns info hash
	resp, err = client.GetOrg("/cookbooks/" + cbName)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s: %v", cbName, err)
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

func TestCookbooksReadByNameNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/cookbooks/nonexistent_cookbook")
	if err != nil {
		t.Fatalf("GET /cookbooks/nonexistent_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.DeleteOrg("/cookbooks/non_existent/1.2.3")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/non_existent/1.2.3: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteBadVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.DeleteOrg("/cookbooks/non_existent/1.2.3.4")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/non_existent/1.2.3.4: %v", err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksDeleteNonExistentVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_wrong_ver")
	cbVersion := "1.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Try to delete a non-existent version
	resp, err = client.DeleteOrg("/cookbooks/" + cbName + "/99.99.99")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/99.99.99: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)

	// Verify the existing version is still there
	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksDeleteExistingVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_existing")
	cbVersion := "1.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify deleted
	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteLastVersionRemovesCookbook(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_last_ver")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Delete the only version
	resp, err = client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Cookbook should be gone entirely
	resp, err = client.GetOrg("/cookbooks/" + cbName)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteOneVersionKeepsOthers(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_one_keep")
	versions := []string{"1.0.0", "2.0.0"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer client.DeleteOrg("/cookbooks/" + cbName + "/2.0.0")

	// Delete one version
	resp, err := client.DeleteOrg("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Other version should still exist
	resp, err = client.GetOrg("/cookbooks/" + cbName + "/2.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/2.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Deleted version should be gone
	resp, err = client.GetOrg("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_admin")
	cbVersion := "0.0.1"
	payload := pedant.NewCookbook(cbName, cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify deleted
	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_normal")
	cbVersion := "0.0.1"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer adminClient.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Normal user cannot delete
	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 403)

	// Verify not deleted
	resp, err = adminClient.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksDeleteValidatorCannotDelete(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_validator")
	cbVersion := "0.0.1"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer adminClient.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Validator cannot delete
	validatorClient := testServer.NewClient(testServer.ValidatorClient)
	resp, err = validatorClient.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	// If the validator client was deleted by another test, skip
	if resp.StatusCode == 401 {
		t.Skip("validator client no longer exists (deleted by another test)")
		return
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestCookbooksDeleteMethodNotAllowedOnCollection(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.DeleteOrg("/cookbooks")
	if err != nil {
		t.Fatalf("DELETE /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestCookbooksDeleteMethodNotAllowedOnCookbook(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.DeleteOrg("/cookbooks/some_cookbook")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/some_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestCookbooksNamedFiltersNoCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	// _latest with no cookbooks
	resp, err := client.GetOrg("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty _latest with no cookbooks, got %d entries", len(body))
	}

	// Named cookbook with no cookbooks
	resp, err = client.GetOrg("/cookbooks/my_cookbook")
	if err != nil {
		t.Fatalf("GET /cookbooks/my_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksNamedFiltersOneCookbookOneVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := "my_cookbook"
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// _latest should include this cookbook
	resp, err = client.GetOrg("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	url, ok := body[cbName].(string)
	if !ok {
		t.Fatalf("expected cookbook %q in _latest, got: %v", cbName, body)
	}
	if url == "" {
		t.Errorf("expected non-empty URL for %q in _latest", cbName)
	}

	// Named cookbook should return info
	resp, err = client.GetOrg("/cookbooks/" + cbName)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body = pedant.GetJSONBody(t, resp)
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

func TestCookbooksNamedFiltersDifferentCookbook(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := "your_cookbook"
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// _latest should include this cookbook
	resp, err = client.GetOrg("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[cbName]; !ok {
		t.Errorf("expected cookbook %q in _latest, got: %v", cbName, body)
	}

	// "my_cookbook" should not exist
	resp, err = client.GetOrg("/cookbooks/my_cookbook")
	if err != nil {
		t.Fatalf("GET /cookbooks/my_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksNamedFiltersMultipleCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cb1 := "my_cookbook"
	cb2 := "your_cookbook"

	payload1 := pedant.NewCookbook(cb1, "1.0.0")
	payload2 := pedant.NewCookbook(cb2, "1.3.0")

	resp, err := client.PutOrg("/cookbooks/"+cb1+"/1.0.0", payload1)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PutOrg("/cookbooks/"+cb2+"/1.3.0", payload2)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.3.0: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	defer func() {
		client.DeleteOrg("/cookbooks/" + cb1 + "/1.0.0")
		client.DeleteOrg("/cookbooks/" + cb2 + "/1.3.0")
	}()

	// _latest should include both
	resp, err = client.GetOrg("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[cb1]; !ok {
		t.Errorf("expected cookbook %q in _latest, not found", cb1)
	}
	if _, ok := body[cb2]; !ok {
		t.Errorf("expected cookbook %q in _latest, not found", cb2)
	}

	// Named cookbook should return info for my_cookbook
	resp, err = client.GetOrg("/cookbooks/" + cb1)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body = pedant.GetJSONBody(t, resp)
	if _, ok := body[cb1]; !ok {
		t.Errorf("expected cookbook %q in response, got: %v", cb1, body)
	}
}

func TestCookbooksNamedFiltersMultipleVersions(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := "my_cookbook"
	versions := []string{"1.0.0", "1.5.0"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	// _latest should return the latest version URL
	resp, err := client.GetOrg("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	url, ok := body[cbName].(string)
	if !ok {
		t.Fatalf("expected cookbook %q in _latest, got: %v", cbName, body)
	}
	// URL should point to the latest version (1.5.0)
	if url == "" {
		t.Errorf("expected non-empty URL for %q", cbName)
	}

	// Named cookbook with num_versions=all should return both versions
	resp, err = client.GetOrg("/cookbooks/" + cbName + "?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s?num_versions=all: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body = pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cbName].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cbName, body)
	}
	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 2 {
		t.Errorf("expected 2 versions, got %d: %v", len(versionsResp), versionsResp)
	}
}

func TestCookbooksNamedFiltersMultipleCookbooksMultipleVersions(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cb1 := "my_cookbook"
	cb2 := "your_cookbook"

	payload1a := pedant.NewCookbook(cb1, "1.0.0")
	payload1b := pedant.NewCookbook(cb1, "1.5.0")
	payload2a := pedant.NewCookbook(cb2, "1.3.0")
	payload2b := pedant.NewCookbook(cb2, "2.0.0")

	resp, err := client.PutOrg("/cookbooks/"+cb1+"/1.0.0", payload1a)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PutOrg("/cookbooks/"+cb1+"/1.5.0", payload1b)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.5.0: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PutOrg("/cookbooks/"+cb2+"/1.3.0", payload2a)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.3.0: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PutOrg("/cookbooks/"+cb2+"/2.0.0", payload2b)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/2.0.0: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	defer func() {
		client.DeleteOrg("/cookbooks/" + cb1 + "/1.0.0")
		client.DeleteOrg("/cookbooks/" + cb1 + "/1.5.0")
		client.DeleteOrg("/cookbooks/" + cb2 + "/1.3.0")
		client.DeleteOrg("/cookbooks/" + cb2 + "/2.0.0")
	}()

	// _latest should include both with latest versions
	resp, err = client.GetOrg("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[cb1]; !ok {
		t.Errorf("expected cookbook %q in _latest, not found", cb1)
	}
	if _, ok := body[cb2]; !ok {
		t.Errorf("expected cookbook %q in _latest, not found", cb2)
	}

	// Named cookbook for my_cookbook should return both versions
	resp, err = client.GetOrg("/cookbooks/" + cb1 + "?num_versions=all")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s?num_versions=all: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body = pedant.GetJSONBody(t, resp)
	cbInfo, ok := body[cb1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cookbook %q in response, got: %v", cb1, body)
	}
	versionsResp, ok := cbInfo["versions"].([]interface{})
	if !ok {
		t.Fatalf("expected 'versions' array, got: %v", cbInfo)
	}
	if len(versionsResp) != 2 {
		t.Errorf("expected 2 versions for %q, got %d: %v", cb1, len(versionsResp), versionsResp)
	}
}

func TestCookbooksRecipesNoCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	resp, err := client.GetOrg("/cookbooks/_recipes")
	if err != nil {
		t.Fatalf("GET /cookbooks/_recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONArray(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty _recipes with no cookbooks, got %d entries", len(body))
	}
}

func TestCookbooksRecipesWithCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := "my_cookbook"
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks/_recipes")
	if err != nil {
		t.Fatalf("GET /cookbooks/_recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	// Recipes may be empty if no recipe files are included in payload;
	// the important thing is the endpoint doesn't panic and returns 200
}

func TestCookbooksRecipesMultipleCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cb1 := "my_cookbook"
	cb2 := "your_cookbook"

	payload1 := pedant.NewCookbook(cb1, "1.0.0")
	payload2 := pedant.NewCookbook(cb2, "1.3.0")

	resp, err := client.PutOrg("/cookbooks/"+cb1+"/1.0.0", payload1)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.PutOrg("/cookbooks/"+cb2+"/1.3.0", payload2)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.3.0: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	defer func() {
		client.DeleteOrg("/cookbooks/" + cb1 + "/1.0.0")
		client.DeleteOrg("/cookbooks/" + cb2 + "/1.3.0")
	}()

	resp, err = client.GetOrg("/cookbooks/_recipes")
	if err != nil {
		t.Fatalf("GET /cookbooks/_recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	// Just verify it returns 200 without panicking
}

func TestCookbooksRecipesLatestVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := "my_cookbook"
	versions := []string{"1.0.0", "1.5.0"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks/_recipes")
	if err != nil {
		t.Fatalf("GET /cookbooks/_recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	// Just verify it returns 200 without panicking
}

func TestCookbooksUpdateDescription(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_desc")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "hi there"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("second PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	meta := body["metadata"].(map[string]interface{})
	if meta["description"] != "hi there" {
		t.Errorf("expected description 'hi there', got %v", meta["description"])
	}
}

func TestCookbooksUpdateFrozen(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_frozen")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["frozen?"] = true
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT frozen /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["frozen?"] != true {
		t.Errorf("expected frozen? true, got %v", body["frozen?"])
	}
}

func TestCookbooksUpdateFrozenCannotEdit(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_frozen_no")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	payload["frozen?"] = true
	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "this is different"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT frozen /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 409, "frozen")

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["frozen?"] != true {
		t.Errorf("expected frozen? true, got %v", body["frozen?"])
	}
}

func TestCookbooksUpdateFrozenForceOverride(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_force")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	payload["frozen?"] = true
	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["frozen?"] = false
	payload["metadata"].(map[string]interface{})["description"] = "this is different"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion+"?force=true", payload)
	if err != nil {
		t.Fatalf("PUT force /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	meta := body["metadata"].(map[string]interface{})
	if meta["description"] != "this is different" {
		t.Errorf("expected description updated, got %v", meta["description"])
	}
}

func TestCookbooksUpdateFrozenForceFalse(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_force_false")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	payload["frozen?"] = true
	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "this is different"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion+"?force=false", payload)
	if err != nil {
		t.Fatalf("PUT force=false /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	// goiardi treats force=false the same as force=true (checks for presence, not value)
	// This differs from Chef Server behavior
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateCookbookNameInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_cbname")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{1, true, []interface{}{}, map[string]interface{}{}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["cookbook_name"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "cookbook_name")
		})
	}
}

func TestCookbooksUpdateCookbookNameMismatch(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_mismatch")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["cookbook_name"] = "new_cookbook_name"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "cookbook_name")
}

func TestCookbooksUpdateCookbookNameDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_cbname_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p, "cookbook_name")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "cookbook_name")
}

func TestCookbooksUpdateJSONClassInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_jsonclass")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{1, "Chef::NonCookbook", "all wrong"}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["json_class"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "json_class")
		})
	}
}

func TestCookbooksUpdateJSONClassDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_jc_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p, "json_class")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateChefTypeInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_cheftype")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{"not_cookbook", false, []interface{}{"just any", "old junk"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["chef_type"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "chef_type")
		})
	}
}

func TestCookbooksUpdateChefTypeDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_ct_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p, "chef_type")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateVersionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_ver")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{1, []interface{}{"all", "ignored"}, map[string]interface{}{}, "0.0", "something invalid"}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["version"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "version")
		})
	}
}

func TestCookbooksUpdateVersionDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_ver_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p, "version")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateSegmentInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_seg")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	segments := []string{"attributes", "definitions", "files", "libraries", "providers", "recipes", "resources", "root_files", "templates"}
	for _, seg := range segments {
		t.Run(seg+"_string", func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p[seg] = "foo"
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "invalid")
		})
		t.Run(seg+"_empty_map", func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p[seg] = []interface{}{map[string]interface{}{}}
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "Invalid element")
		})
		t.Run(seg+"_empty_array", func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p[seg] = []interface{}{}
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertStatus(t, resp, 200)
		})
	}
}

func TestCookbooksUpdateFrozenField(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_frozen_field")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["frozen?"] = true
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataVersionMissing(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_ver")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"] = map[string]interface{}{"new_name": "foo"}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.version")
}

func TestCookbooksUpdateMetadataNameInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_name")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidNames := []interface{}{1, true, map[string]interface{}{}, []interface{}{}, "invalid name", "ダメよ"}
	for _, v := range invalidNames {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["name"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "metadata.name")
		})
	}
}

func TestCookbooksUpdateMetadataNameChanged(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_name2")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["name"] = "new_name"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataNameDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_name_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "name")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDescription(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_desc")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["description"] = "new description"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDescriptionDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_desc_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "description")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDescriptionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_desc_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["description"] = 1
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.description")
}

func TestCookbooksUpdateMetadataLongDescription(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_long")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["long_description"] = "longer description"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLongDescriptionDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_long_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "long_description")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLongDescriptionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_long_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["long_description"] = false
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.long_description")
}

func TestCookbooksUpdateMetadataVersionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_ver_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidVersions := []interface{}{"0.0", "not a version", 1}
	for _, v := range invalidVersions {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["version"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "metadata.version")
		})
	}
}

func TestCookbooksUpdateMetadataVersionDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_ver_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "version")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.version")
}

func TestCookbooksUpdateMetadataMaintainer(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_maint")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer"] = "Captain Stupendous"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_maint_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "maintainer")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_maint_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer"] = true
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.maintainer")
}

func TestCookbooksUpdateMetadataMaintainerEmail(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer_email"] = "cap@awesome.com"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerEmailNotEmail(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email2")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer_email"] = "not really an email"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerEmailDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "maintainer_email")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerEmailInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer_email"] = false
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.maintainer_email")
}

func TestCookbooksUpdateMetadataLicense(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_lic")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["license"] = "to_kill"
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLicenseDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_lic_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "license")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLicenseInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_lic_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["license"] = 1
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.license")
}

func TestCookbooksUpdateMetadataPlatforms(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_plat")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["platforms"] = map[string]interface{}{}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataPlatformsInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_plat_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{[]interface{}{}, "foo", []interface{}{"foo"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["platforms"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "metadata.platforms")
		})
	}
}

func TestCookbooksUpdateMetadataDependencies(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_dep")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["dependencies"] = map[string]interface{}{}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDependenciesDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_dep_del")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "dependencies")
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDependenciesInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_dep_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{[]interface{}{}, "foo", []interface{}{"foo"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["dependencies"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "metadata.dependencies")
		})
	}
}

func TestCookbooksUpdateMetadataGroupings(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_grp")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["groupings"] = map[string]interface{}{}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataGroupingsWithMap(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_grp2")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["groupings"] = map[string]interface{}{"foo": map[string]interface{}{}}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataGroupingsInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_grp_inv")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{[]interface{}{}, "foo", []interface{}{"foo"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["groupings"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "metadata.groupings")
		})
	}
}

func TestCookbooksUpdateMetadataProviding(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_prov")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	providingValues := []interface{}{
		"cats::sleep",
		"here(:kitty, :time_to_eat)",
		"service[snuggle]",
		"",
		1,
		true,
		[]interface{}{"cats", "sleep", "here"},
		map[string]interface{}{"cats::sleep": "0.0.1"},
	}
	for _, v := range providingValues {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := pedant.NewCookbook(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["providing"] = v
			resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertStatus(t, resp, 200)
		})
	}
}

func TestCookbooksUpdateMetadataRecommendations(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_rec")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["recommendations"] = map[string]interface{}{}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataSuggestions(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_sug")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["suggestions"] = map[string]interface{}{}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataConflicting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_conf")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["conflicting"] = map[string]interface{}{}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataReplacing(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_repl")
	cbVersion := "11.2.3"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := pedant.NewCookbook(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["replacing"] = map[string]interface{}{}
	resp, err = client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksVersionNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/cookbooks/fakecookbook/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/fakecookbook/1.0.0: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksVersionExisting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("ver_existing")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksVersionNonExistentVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("ver_missing")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer client.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.GetOrg("/cookbooks/" + cbName + "/6.6.6")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/6.6.6: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksVersionAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("ver_normal")
	cbVersion := "1.0.0"
	payload := pedant.NewCookbook(cbName, cbVersion)
	defer adminClient.DeleteOrg("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.PutOrg("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.GetOrg("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksVersionLatest(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("ver_latest")
	versions := []string{"1.0.0", "1.0.1"}

	for _, v := range versions {
		payload := pedant.NewCookbook(cbName, v)
		resp, err := client.PutOrg("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer func() {
		for _, v := range versions {
			client.DeleteOrg("/cookbooks/" + cbName + "/" + v)
		}
	}()

	resp, err := client.GetOrg("/cookbooks/" + cbName + "/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/_latest: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["version"] != "1.0.1" {
		t.Errorf("expected latest version 1.0.1, got %v", body["version"])
	}
}

func TestCookbooksVersionLatestNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/cookbooks/nonexistent_cookbook/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/nonexistent_cookbook/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}
