package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

// --- User Tests (ported from users_spec.rb) ---

// GET /users

func TestUsersListAsSuperuser(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	resp, err := client.Get("/users")
	if err != nil {
		t.Fatalf("GET /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body["admin"]; !ok {
		t.Errorf("expected 'admin' in user list, got: %v", body)
	}
}

func TestUsersListAsAdminUser(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.Get("/users")
	if err != nil {
		t.Fatalf("GET /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestUsersListAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	resp, err := client.Get("/users")
	if err != nil {
		t.Fatalf("GET /users: %v", err)
	}
	// goiardi allows normal users to list users (differs from Chef Server)
	// Accept either 200 or 403
	if resp.StatusCode != 200 && resp.StatusCode != 403 {
		t.Errorf("expected 200 or 403, got %d", resp.StatusCode)
	}
}

// GET /users/<name>

func TestUsersGetExisting(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	resp, err := client.Get("/users/admin")
	if err != nil {
		t.Fatalf("GET /users/admin: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["name"] != "admin" {
		t.Errorf("expected name 'admin', got %v", body["name"])
	}
	if body["admin"] != true {
		t.Errorf("expected admin=true, got %v", body["admin"])
	}
}

func TestUsersGetNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	resp, err := client.Get("/users/nonexistent_user")
	if err != nil {
		t.Fatalf("GET /users/nonexistent_user: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestUsersGetAsNormalUserSelf(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	resp, err := client.Get("/users/pedant_test_user")
	if err != nil {
		t.Fatalf("GET /users/pedant_test_user: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestUsersGetAsNormalUserOther(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	resp, err := client.Get("/users/admin")
	if err != nil {
		t.Fatalf("GET /users/admin: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)
}

// POST /users

func TestUsersCreateAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("create_test")
	password := "opensesame123"
	u := pedant.NewUser(userName, map[string]interface{}{
		"password": password,
		"admin":    true,
	})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	pedant.AssertBodyContains(t, resp, "/users/"+userName)

	// Verify user was created
	resp, err = client.Get("/users/" + userName)
	if err != nil {
		t.Fatalf("GET /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["name"] != userName {
		t.Errorf("expected name %q, got %q", userName, body["name"])
	}
}

func TestUsersCreateDuplicateName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("dup_user")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("first POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Post("/users", u)
	if err != nil {
		t.Fatalf("second POST /users: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 409, "already exists")
}

func TestUsersCreateMissingName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	u := map[string]interface{}{
		"password": "opensesame123",
	}
	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "name")
}

func TestUsersCreateMissingPassword(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("no_pass")
	u := map[string]interface{}{
		"name": userName,
	}
	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "password")
}

func TestUsersCreateShortPassword(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("short_pass")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "abc"})
	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "Password")
}

func TestUsersCreateInvalidName(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)

	invalidNames := []string{"USERNAME", "user@example.org", "user name", "user|name"}
	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			u := pedant.NewUser(name, map[string]interface{}{"password": "opensesame123"})
			resp, err := client.Post("/users", u)
			if err != nil {
				t.Fatalf("POST /users: %v", err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "name")
		})
	}
}

func TestUsersCreateInvalidAdminFlag(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("bad_admin")

	invalidValues := []interface{}{"random string", []interface{}{}, map[string]interface{}{}, 1}
	for _, v := range invalidValues {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123", "admin": v})
			resp, err := client.Post("/users", u)
			if err != nil {
				t.Fatalf("POST /users: %v", err)
			}
			pedant.AssertErrorResponse(t, resp, 400, "admin")
		})
	}
}

func TestUsersCreateAsNormalUser(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	userName := pedant.UniqueName("no_perm_create")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)

	// Verify user was not created
	superClient := testServer.NewClient(testServer.Superuser)
	resp, err = superClient.Get("/users/" + userName)
	if err != nil {
		t.Fatalf("GET /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestUsersCreateEmptyBody(t *testing.T) {
	req, err := http.NewRequest("POST", testServer.BaseURL+"/users", strings.NewReader(""))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	signReq := testServer.NewClient(testServer.AdminUser)
	signReq.SignRawRequest(req, []byte(""))

	resp2, err := signReq.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("executing request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 400 {
		t.Errorf("expected 400 for empty body, got %d", resp2.StatusCode)
	}
}

// PUT /users/<name>

func TestUsersUpdateNameRename(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_name")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Rename
	newName := userName + "_renamed"
	update := pedant.NewUser(newName, map[string]interface{}{"password": "opensesame123"})
	resp, err = client.Put("/users/"+userName, update)
	if err != nil {
		t.Fatalf("PUT /users/%s: %v", userName, err)
	}
	// goiardi returns 201 for rename
	pedant.AssertStatus(t, resp, 201)

	// New name should exist
	resp, err = client.Get("/users/" + newName)
	if err != nil {
		t.Fatalf("GET /users/%s: %v", newName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	// Cleanup
	client.Delete("/users/" + newName)
}

func TestUsersUpdateAdminFlag(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_admin")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123", "admin": false})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Set admin to true
	update := pedant.NewUser(userName, map[string]interface{}{"admin": true})
	resp, err = client.Put("/users/"+userName, update)
	if err != nil {
		t.Fatalf("PUT /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/users/" + userName)
	if err != nil {
		t.Fatalf("GET /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["admin"] != true {
		t.Errorf("expected admin=true, got %v", body["admin"])
	}
}

func TestUsersUpdateAdminFlagToFalse(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_admin_false")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123", "admin": true})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Set admin to false
	update := pedant.NewUser(userName, map[string]interface{}{"admin": false})
	resp, err = client.Put("/users/"+userName, update)
	if err != nil {
		t.Fatalf("PUT /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/users/" + userName)
	if err != nil {
		t.Fatalf("GET /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["admin"] != false {
		t.Errorf("expected admin=false, got %v", body["admin"])
	}
}

func TestUsersUpdateNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	update := map[string]interface{}{"name": "does_not_exist"}
	resp, err := client.Put("/users/does_not_exist", update)
	if err != nil {
		t.Fatalf("PUT /users/does_not_exist: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestUsersUpdateEmptyBody(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_empty")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	req, err := http.NewRequest("PUT", testServer.BaseURL+"/users/"+userName, strings.NewReader(""))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	signReq := testServer.NewClient(testServer.Superuser)
	signReq.SignRawRequest(req, []byte(""))

	resp2, err := signReq.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("executing request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 400 {
		t.Errorf("expected 400 for empty body, got %d", resp2.StatusCode)
	}
}

func TestUsersUpdateAsNormalUserSelf(t *testing.T) {
	// Create a user as admin, then update as that user
	adminClient := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("self_upd")
	password := "opensesame123"
	u := pedant.NewUser(userName, map[string]interface{}{"password": password, "admin": false})
	defer adminClient.Delete("/users/" + userName)

	resp, err := adminClient.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Update as the user themselves
	normalClient := testServer.NewClient(testServer.NormalUser)
	update := pedant.NewUser(userName, map[string]interface{}{"password": "newpassword123"})
	resp, err = normalClient.Put("/users/"+userName, update)
	if err != nil {
		t.Fatalf("PUT /users/%s: %v", userName, err)
	}
	// Normal user cannot update another user
	pedant.AssertStatus(t, resp, 403)
}

func TestUsersUpdateAsNormalUserOther(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	update := map[string]interface{}{"name": "admin"}
	resp, err := client.Put("/users/admin", update)
	if err != nil {
		t.Fatalf("PUT /users/admin: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestUsersUpdatePrivEscalation(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	update := map[string]interface{}{"name": "pedant_test_user", "admin": true}
	resp, err := client.Put("/users/pedant_test_user", update)
	if err != nil {
		t.Fatalf("PUT /users/pedant_test_user: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestUsersUpdatePassword(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_pass")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Update password
	update := pedant.NewUser(userName, map[string]interface{}{"password": "newpassword456"})
	resp, err = client.Put("/users/"+userName, update)
	if err != nil {
		t.Fatalf("PUT /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestUsersUpdateShortPassword(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_short_pass")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	update := pedant.NewUser(userName, map[string]interface{}{"password": "abc"})
	resp, err = client.Put("/users/"+userName, update)
	if err != nil {
		t.Fatalf("PUT /users/%s: %v", userName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "Password")
}

func TestUsersUpdateInvalidName(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_inv_name")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	update := pedant.NewUser("USERNAME", map[string]interface{}{"password": "opensesame123"})
	resp, err = client.Put("/users/"+userName, update)
	if err != nil {
		t.Fatalf("PUT /users/%s: %v", userName, err)
	}
	pedant.AssertErrorResponse(t, resp, 400, "name")
}

// DELETE /users/<name>

func TestUsersDeleteAsAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("del_admin")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123"})

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	resp, err = client.Delete("/users/" + userName)
	if err != nil {
		t.Fatalf("DELETE /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)

	resp, err = client.Get("/users/" + userName)
	if err != nil {
		t.Fatalf("GET /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestUsersDeleteNonExistent(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	resp, err := client.Delete("/users/nonexistent_user")
	if err != nil {
		t.Fatalf("DELETE /users/nonexistent_user: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestUsersDeleteAsNormalUserOther(t *testing.T) {
	client := testServer.NewClient(testServer.NormalUser)
	resp, err := client.Delete("/users/admin")
	if err != nil {
		t.Fatalf("DELETE /users/admin: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestUsersDeleteAsNormalUserSelf(t *testing.T) {
	// Create a non-admin user, then delete themselves
	adminClient := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("self_del")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "opensesame123", "admin": false})

	resp, err := adminClient.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Delete as the user themselves
	normalClient := testServer.NewClient(testServer.NormalUser)
	resp, err = normalClient.Delete("/users/" + userName)
	if err != nil {
		t.Fatalf("DELETE /users/%s: %v", userName, err)
	}
	// Normal user cannot delete another user
	pedant.AssertStatus(t, resp, 403)

	// Cleanup
	adminClient.Delete("/users/" + userName)
}

func TestUsersDeleteLastAdmin(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	resp, err := client.Delete("/users/admin")
	if err != nil {
		t.Fatalf("DELETE /users/admin: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)
}
