package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- Cookbook Delete Tests ---

func TestCookbooksDeleteNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Delete("/cookbooks/non_existent/1.2.3")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/non_existent/1.2.3: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteBadVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Delete("/cookbooks/non_existent/1.2.3.4")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/non_existent/1.2.3.4: %v", err)
	}
	pedant.AssertStatus(t, resp, 400)
}

func TestCookbooksDeleteNonExistentVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_wrong_ver")
	cbVersion := "1.2.3"
	payload := newCookbookPayload(cbName, cbVersion)
	defer client.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Try to delete a non-existent version
	resp, err = client.Delete("/cookbooks/" + cbName + "/99.99.99")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/99.99.99: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)

	// Verify the existing version is still there
	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksDeleteExistingVersion(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_existing")
	cbVersion := "1.2.3"
	payload := newCookbookPayload(cbName, cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Delete("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify deleted
	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteLastVersionRemovesCookbook(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_last_ver")
	cbVersion := "1.0.0"
	payload := newCookbookPayload(cbName, cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Delete the only version
	resp, err = client.Delete("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Cookbook should be gone entirely
	resp, err = client.Get("/cookbooks/" + cbName)
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
		payload := newCookbookPayload(cbName, v)
		resp, err := client.Put("/cookbooks/"+cbName+"/"+v, payload)
		if err != nil {
			t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, v, err)
		}
		pedant.AssertStatus(t, resp, 201)
	}
	defer client.Delete("/cookbooks/" + cbName + "/2.0.0")

	// Delete one version
	resp, err := client.Delete("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Other version should still exist
	resp, err = client.Get("/cookbooks/" + cbName + "/2.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/2.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Deleted version should be gone
	resp, err = client.Get("/cookbooks/" + cbName + "/1.0.0")
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/1.0.0: %v", cbName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_admin")
	cbVersion := "0.0.1"
	payload := newCookbookPayload(cbName, cbVersion)

	resp, err := client.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Delete("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Verify deleted
	resp, err = client.Get("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestCookbooksDeleteAsNormalUser(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_normal")
	cbVersion := "0.0.1"
	payload := newCookbookPayload(cbName, cbVersion)
	defer adminClient.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Normal user cannot delete
	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.Delete("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("DELETE /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 403)

	// Verify not deleted
	resp, err = adminClient.Get("/cookbooks/" + cbName + "/" + cbVersion)
	if err != nil {
		t.Fatalf("GET /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestCookbooksDeleteValidatorCannotDelete(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	cbName := pedant.UniqueName("del_validator")
	cbVersion := "0.0.1"
	payload := newCookbookPayload(cbName, cbVersion)
	defer adminClient.Delete("/cookbooks/" + cbName + "/" + cbVersion)

	resp, err := adminClient.Put("/cookbooks/"+cbName+"/"+cbVersion, payload)
	if err != nil {
		t.Fatalf("PUT /cookbooks/%s/%s: %v", cbName, cbVersion, err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Validator cannot delete
	validatorClient := testServer.NewClient(testServer.ValidatorClient)
	resp, err = validatorClient.Delete("/cookbooks/" + cbName + "/" + cbVersion)
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
	resp, err := client.Delete("/cookbooks")
	if err != nil {
		t.Fatalf("DELETE /cookbooks: %v", err)
	}
	pedant.AssertStatus(t, resp, 405)
}

func TestCookbooksDeleteMethodNotAllowedOnCookbook(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Delete("/cookbooks/some_cookbook")
	if err != nil {
		t.Fatalf("DELETE /cookbooks/some_cookbook: %v", err)
	}
	pedant.AssertStatus(t, resp, 405)
}
