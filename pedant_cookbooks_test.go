package main

import (
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

func TestCookbooksRecipes(t *testing.T) {
	// _recipes endpoint panics when there are no cookbooks with versions
	// This is a known goiardi bug
	t.Skip("_recipes endpoint panics on empty cookbook list (known bug)")

	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("recipes_cb")
	payload := newCookbookPayload(cbName, "1.0.0")
	defer client.Delete("/cookbooks/" + cbName + "/1.0.0")

	resp, err := client.Put("/cookbooks/"+cbName+"/1.0.0", payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks/_recipes")
	if err != nil {
		t.Fatalf("GET /cookbooks/_recipes: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}
