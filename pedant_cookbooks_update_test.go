package main

import (
	"fmt"
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Cookbook Update Tests ---

func TestCookbooksUpdateDescription(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_desc")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "hi there"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("second PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["frozen?"] = true
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT frozen /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	payload["frozen?"] = true
	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "this is different"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT frozen /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 409, "frozen")

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	payload["frozen?"] = true
	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["frozen?"] = false
	payload["metadata"].(map[string]interface{})["description"] = "this is different"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion+"?force=true", payload)
	if err != nil {
		t.Fatalf("PUT force /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	payload["frozen?"] = true
	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	payload["metadata"].(map[string]interface{})["description"] = "this is different"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion+"?force=false", payload)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{1, true, []interface{}{}, map[string]interface{}{}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["cookbook_name"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["cookbook_name"] = "new_cookbook_name"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "cookbook_name")
}

func TestCookbooksUpdateCookbookNameDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_cbname_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p, "cookbook_name")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "cookbook_name")
}

func TestCookbooksUpdateJSONClassInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_jsonclass")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{1, "Chef::NonCookbook", "all wrong"}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["json_class"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p, "json_class")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateChefTypeInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_cheftype")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{"not_cookbook", false, []interface{}{"just any", "old junk"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["chef_type"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p, "chef_type")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateVersionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_ver")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{1, []interface{}{"all", "ignored"}, map[string]interface{}{}, "0.0", "something invalid"}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["version"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p, "version")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateSegmentInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_seg")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	segments := []string{"attributes", "definitions", "files", "libraries", "providers", "recipes", "resources", "root_files", "templates"}
	for _, seg := range segments {
		t.Run(seg+"_string", func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p[seg] = "foo"
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "invalid")
		})
		t.Run(seg+"_empty_map", func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p[seg] = []interface{}{map[string]interface{}{}}
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
			if err != nil {
				t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "Invalid element")
		})
		t.Run(seg+"_empty_array", func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p[seg] = []interface{}{}
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["frozen?"] = true
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataVersionMissing(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_ver")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"] = map[string]interface{}{"new_name": "foo"}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.version")
}

func TestCookbooksUpdateMetadataNameInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_name")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidNames := []interface{}{1, true, map[string]interface{}{}, []interface{}{}, "invalid name", "ダメよ"}
	for _, v := range invalidNames {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["name"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["name"] = "new_name"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataNameDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_name_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "name")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDescription(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_desc")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["description"] = "new description"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDescriptionDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_desc_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "description")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDescriptionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_desc_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["description"] = 1
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.description")
}

func TestCookbooksUpdateMetadataLongDescription(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_long")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["long_description"] = "longer description"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLongDescriptionDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_long_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "long_description")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLongDescriptionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_long_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["long_description"] = false
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.long_description")
}

func TestCookbooksUpdateMetadataVersionInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_ver_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidVersions := []interface{}{"0.0", "not a version", 1}
	for _, v := range invalidVersions {
		t.Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["version"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "version")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.version")
}

func TestCookbooksUpdateMetadataMaintainer(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_maint")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer"] = "Captain Stupendous"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_maint_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "maintainer")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_maint_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer"] = true
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.maintainer")
}

func TestCookbooksUpdateMetadataMaintainerEmail(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer_email"] = "cap@awesome.com"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerEmailNotEmail(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email2")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer_email"] = "not really an email"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerEmailDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "maintainer_email")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataMaintainerEmailInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_email_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["maintainer_email"] = false
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.maintainer_email")
}

func TestCookbooksUpdateMetadataLicense(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_lic")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["license"] = "to_kill"
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLicenseDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_lic_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "license")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataLicenseInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_lic_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["license"] = 1
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "metadata.license")
}

func TestCookbooksUpdateMetadataPlatforms(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_plat")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["platforms"] = map[string]interface{}{}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataPlatformsInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_plat_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{[]interface{}{}, "foo", []interface{}{"foo"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["platforms"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["dependencies"] = map[string]interface{}{}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDependenciesDeleted(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_dep_del")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	delete(p["metadata"].(map[string]interface{}), "dependencies")
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataDependenciesInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_dep_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{[]interface{}{}, "foo", []interface{}{"foo"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["dependencies"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["groupings"] = map[string]interface{}{}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataGroupingsWithMap(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_grp2")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["groupings"] = map[string]interface{}{"foo": map[string]interface{}{}}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataGroupingsInvalid(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_grp_inv")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	invalidValues := []interface{}{[]interface{}{}, "foo", []interface{}{"foo"}}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			p := newCookbookPayload(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["groupings"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
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
			p := newCookbookPayload(cbName, cbVersion)
			p["metadata"].(map[string]interface{})["providing"] = v
			resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["recommendations"] = map[string]interface{}{}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataSuggestions(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_sug")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["suggestions"] = map[string]interface{}{}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataConflicting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_conf")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["conflicting"] = map[string]interface{}{}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksUpdateMetadataReplacing(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("upd_meta_repl")
	cbVersion := "11.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	p := newCookbookPayload(cbName, cbVersion)
	p["metadata"].(map[string]interface{})["replacing"] = map[string]interface{}{}
	resp, err = client.Put("/cookbooks/"+cbName+"/"+cbVersion, p)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}
