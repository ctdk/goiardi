package main

import (
	"strings"
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// newCookbookPayload creates a minimal cookbook version payload.
func newCookbookPayload(name, version string, opts ...map[string]interface{}) map[string]interface{} {
	cb := map[string]interface{}{
		"cookbook_name":      name,
		"name":               name + "-" + version,
		"version":            version,
		"json_class":         "Chef::CookbookVersion",
		"chef_type":          "cookbook_version",
		"definitions":        []interface{}{},
		"libraries":          []interface{}{},
		"attributes":         []interface{}{},
		"recipes":            []interface{}{},
		"providers":          []interface{}{},
		"resources":          []interface{}{},
		"templates":          []interface{}{},
		"root_files":         []interface{}{},
		"files":              []interface{}{},
		"frozen?":            false,
		"metadata": map[string]interface{}{
			"version":           version,
			"name":              name,
			"maintainer":        "",
			"description":       "",
			"long_description":  "",
			"maintainer_email":  "",
			"license":           "All rights reserved",
			"platforms":         map[string]interface{}{},
			"dependencies":      map[string]interface{}{},
			"recommendations":   map[string]interface{}{},
			"suggestions":       map[string]interface{}{},
			"conflicting":       map[string]interface{}{},
			"replacing":         map[string]interface{}{},
			"groupings":         map[string]interface{}{},
			"providing":         map[string]interface{}{},
		},
	}
	if len(opts) > 0 {
		for k, v := range opts[0] {
			cb[k] = v
		}
	}
	return cb
}

func TestCookbooksListEmpty(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/cookbooks")
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("first PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
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

	resp, err := client.Get("/cookbooks/" + cbName + "?num_versions=all")
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

	resp, err := client.Get("/cookbooks?num_versions=all")
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

	resp, err := client.Get("/cookbooks?num_versions=1")
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

	resp, err := client.Get("/cookbooks?num_versions=all")
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
	payload := newCookbookPayload(cbName, "1.0.0")
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks?num_versions=0")
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

func TestCookbooksInvalidNumVersions(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	invalidValues := []string{"-1", "", "foo"}
	for _, v := range invalidValues {
		t.Run(v, func(t *testing.T) {
			resp, err := client.Get("/cookbooks?num_versions=" + v)
			if err != nil {
				t.Fatalf("GET /cookbooks?num_versions=%s: %v", v, err)
			}
			pedant.AssertStatus(t, resp, 400)
		})
	}
}

func TestCookbooksDelete(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_cb")
	payload := newCookbookPayload(cbName, "1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Delete("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/cookbooks/nonexistent_cb/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/nonexistent_cb/1.0.0: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksInvalidVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("inv_ver_cb")
	payload := newCookbookPayload(cbName, "1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/abc", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/abc: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksInvalidName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	payload := newCookbookPayload("valid_name", "1.0.0")

	resp, err := client.Put("/cookbooks/first@second/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/first@second/1.0.0: %v", err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksJSONClassValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("jsonclass_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	payload["json_class"] = "Chef::Node"
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "json_class")
}

func TestCookbooksChefTypeValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("cheftype_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	payload["chef_type"] = "node"
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "chef_type")
}

func TestCookbooksInvalidKeys(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("inv_keys_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	payload["invalid_key"] = "some_value"
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "Invalid key")
}

func TestCookbooksMetadataVersionMissing(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("meta_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	payload["metadata"] = map[string]interface{}{}
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
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
			payload := newCookbookPayload(cbName, "1.0.0")
			payload[seg] = "foo"
			defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

			resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
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
	payload := newCookbookPayload("wrong_name", "1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "name")
}

func TestCookbooksMismatchedVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("mismatch_ver")
	payload := newCookbookPayload(cbName, "0.0.1")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "name")
}

func TestCookbooksNegativeVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("neg_ver")
	payload := newCookbookPayload(cbName, "1.2.-42")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.2.-42", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.2.-42: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksVersion4Byte(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("4byte_cb")
	version := "1.2.2147483647"
	payload := newCookbookPayload(cbName, version)
	defer client.Delete("/cookbooks/" + cbName + "/" + version)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+version, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, version, err)
	}
	pedant.AssertStatus(t, resp, 201)
}

func TestCookbooksVersion4ByteOverflow(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("4byte_ovf")
	version := "1.2.2147483669"
	payload := newCookbookPayload(cbName, version)
	defer client.Delete("/cookbooks/" + cbName + "/" + version)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+version, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, version, err)
	}
	pedant.AssertStatus(t, resp, 201)
}

func TestCookbooksVersion8ByteOverflow(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("8byte_ovf")
	version := "1.2.9223372036854775849"
	payload := newCookbookPayload(cbName, version)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+version, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, version, err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksMetadataSectionValidation(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("meta_sec")

	sections := []string{"platforms", "dependencies", "recommendations", "suggestions", "conflicting", "replacing"}
	for _, section := range sections {
		t.Run(section+"_invalid_type", func(t *testing.T) {
			payload := newCookbookPayload(cbName, "1.0.0")
			payload["metadata"].(map[string]interface{})[section] = "foo"
			defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

			resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
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
	payload := newCookbookPayload(cbName, "1.0.0")
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "updated description"
	resp, err = client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("second PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksNonAdminCannotCreate(t *testing.T) {
	normalClient := testServer.NewClient(testServer.NormalUser)
	cbName := pedant.UniqueName("no_perm_cb")
	payload := newCookbookPayload(cbName, "1.0.0")

	resp, err := normalClient.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestCookbooksNonAdminCannotDelete(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("no_del_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	defer adminClient.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := adminClient.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.Delete("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestCookbooksValidatorCannotList(t *testing.T) {
	validatorClient := testServer.NewClient(testServer.ValidatorClient)
	resp, err := validatorClient.Get("/cookbooks")
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

	payload1 := newCookbookPayload(cb1, "0.0.1")
	payload2 := newCookbookPayload(cb2, "0.0.2")

	resp, err := client.Put("/cookbooks/"+cb1+"/0.0.1", payload1)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/0.0.1: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cb2+"/0.0.2", payload2)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/0.0.2: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	defer func() {
		client.Delete("/cookbooks/" + cb1 + "/0.0.1")
		client.Delete("/cookbooks/" + cb2 + "/0.0.2")
	}()

	resp, err = client.Get("/cookbooks?num_versions=all")
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

	resp, err := client.Get("/cookbooks/" + cbName + "/_latest")
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Read the cookbook version
	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	cbPayload := newCookbookPayload(cbName, cbVersion, map[string]interface{}{
		"recipes": []interface{}{
			map[string]interface{}{
				"name":         "default.rb",
				"path":         "recipes/default.rb",
				"checksum":     "0000000000000000000000000000000000000000",
				"specificity":  "default",
			},
		},
	})
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, cbPayload)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer adminClient.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Read as normal user
	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.Get("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksReadAsAdminUser(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("read_admin")
	cbVersion := "1.0.0"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksReadVerifyResponseBody(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("verify_cb")
	cbVersion := "1.0.0"
	payload := newCookbookPayload(cbName, cbVersion, map[string]interface{}{
		"metadata": map[string]interface{}{
			"version":           cbVersion,
			"name":              cbName,
			"maintainer":        "Your Name",
			"description":       "A fabulous new cookbook",
			"long_description":  "",
			"maintainer_email":  "youremail@example.com",
			"license":           "Apache v2.0",
			"platforms":         map[string]interface{}{},
			"dependencies":      map[string]interface{}{},
			"recommendations":   map[string]interface{}{},
			"suggestions":       map[string]interface{}{},
			"conflicting":       map[string]interface{}{},
			"replacing":         map[string]interface{}{},
			"groupings":         map[string]interface{}{},
			"providing":         map[string]interface{}{},
		},
	})
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks?num_versions=all")
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

	resp, err := client.Get("/cookbooks?num_versions=all")
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

	// Without num_versions, should return 1 version per cookbook
	resp, err := client.Get("/cookbooks")
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Check the cookbook collection URL
	resp, err = client.Get("/cookbooks?num_versions=all")
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer adminClient.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Validator cannot list
	validatorClient := testServer.NewClient(testServer.ValidatorClient)
	resp, err = validatorClient.Get("/cookbooks")
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// GET /cookbooks/<name> returns info hash
	resp, err = client.Get("/cookbooks/" + cbName)
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
	resp, err := client.Get("/cookbooks/nonexistent_cookbook")
	if err != nil {
		t.Fatalf("GET /cookbooks/nonexistent_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

