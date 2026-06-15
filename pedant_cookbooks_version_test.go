package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Cookbook Version Tests ---

func TestCookbooksVersionNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/cookbooks/fakecookbook/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/fakecookbook/1.0.0: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksVersionExisting(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("ver_existing")
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

func TestCookbooksVersionNonExistentVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("ver_missing")
	cbVersion := "1.0.0"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Get("/cookbooks/" + cbName + "/6.6.6")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/6.6.6: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksVersionAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("ver_normal")
	cbVersion := "1.0.0"
	payload := newCookbookPayload(cbName, cbVersion)
	defer adminClient.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.Get("/cookbooks/" + cbName + "/" + cbVersion)
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
	if body["version"] != "1.0.1" {
		t.Errorf("expected latest version 1.0.1, got %v", body["version"])
	}
}

func TestCookbooksVersionLatestNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/cookbooks/nonexistent_cookbook/_latest")
	if err != nil {
		t.Fatalf("GET /cookbooks/nonexistent_cookbook/_latest: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}
