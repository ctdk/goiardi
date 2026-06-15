package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Cookbook Named Filters Tests ---

func TestCookbooksNamedFiltersNoCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	// _latest with no cookbooks
	resp, err := client.Get("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty _latest with no cookbooks, got %d entries", len(body))
	}

	// Named cookbook with no cookbooks
	resp, err = client.Get("/cookbooks/my_cookbook")
	if err != nil {
		t.Fatalf("GET /cookbooks/my_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksNamedFiltersOneCookbookOneVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := "my_cookbook"
	cbVersion := "1.0.0"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// _latest should include this cookbook
	resp, err = client.Get("/cookbooks/_latest")
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
	resp, err = client.Get("/cookbooks/" + cbName)
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
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// _latest should include this cookbook
	resp, err = client.Get("/cookbooks/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body[cbName]; !ok {
		t.Errorf("expected cookbook %q in _latest, got: %v", cbName, body)
	}

	// "my_cookbook" should not exist
	resp, err = client.Get("/cookbooks/my_cookbook")
	if err != nil {
		t.Fatalf("GET /cookbooks/my_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksNamedFiltersMultipleCookbooks(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cb1 := "my_cookbook"
	cb2 := "your_cookbook"

	payload1 := newCookbookPayload(cb1, "1.0.0")
	payload2 := newCookbookPayload(cb2, "1.3.0")

	resp, err := client.Put("/cookbooks/"+cb1+"/1.0.0", payload1)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cb2+"/1.3.0", payload2)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.3.0: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	defer func() {
		client.Delete("/cookbooks/" + cb1 + "/1.0.0")
		client.Delete("/cookbooks/" + cb2 + "/1.3.0")
	}()

	// _latest should include both
	resp, err = client.Get("/cookbooks/_latest")
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
	resp, err = client.Get("/cookbooks/" + cb1)
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

	// _latest should return the latest version URL
	resp, err := client.Get("/cookbooks/_latest")
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
	resp, err = client.Get("/cookbooks/" + cbName + "?num_versions=all")
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

	payload1a := newCookbookPayload(cb1, "1.0.0")
	payload1b := newCookbookPayload(cb1, "1.5.0")
	payload2a := newCookbookPayload(cb2, "1.3.0")
	payload2b := newCookbookPayload(cb2, "2.0.0")

	resp, err := client.Put("/cookbooks/"+cb1+"/1.0.0", payload1a)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.0.0: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cb1+"/1.5.0", payload1b)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.5.0: %v", cb1, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cb2+"/1.3.0", payload2a)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/1.3.0: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Put("/cookbooks/"+cb2+"/2.0.0", payload2b)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/2.0.0: %v", cb2, err)
	}
	pedant.AssertStatus(t, resp, 201)

	defer func() {
		client.Delete("/cookbooks/" + cb1 + "/1.0.0")
		client.Delete("/cookbooks/" + cb1 + "/1.5.0")
		client.Delete("/cookbooks/" + cb2 + "/1.3.0")
		client.Delete("/cookbooks/" + cb2 + "/2.0.0")
	}()

	// _latest should include both with latest versions
	resp, err = client.Get("/cookbooks/_latest")
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
	resp, err = client.Get("/cookbooks/" + cb1 + "?num_versions=all")
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
