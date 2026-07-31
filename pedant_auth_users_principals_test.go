package main

import (
	"testing"

	"github.com/ctdk/goiardi/pedant"
)

func TestAuthenticateUserSuccess(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("auth_user")
	password := "test_password_123"
	u := pedant.NewUser(userName, map[string]interface{}{"password": password})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Authenticate
	authPayload := map[string]interface{}{
		"name":     userName,
		"password": password,
	}
	resp, err = client.Post("/authenticate_user", authPayload)
	if err != nil {
		t.Fatalf("POST /authenticate_user: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	userBody, ok := body["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'user' object in response, got: %v", body)
	}
	if userBody["username"] != userName {
		t.Errorf("expected username %q, got %v", userName, userBody["username"])
	}
	if body["status"] != "linked" {
		t.Errorf("expected status 'linked', got %v", body["status"])
	}
}

func TestAuthenticateUserWrongPassword(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("auth_fail")
	u := pedant.NewUser(userName, map[string]interface{}{"password": "correct_password"})
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	authPayload := map[string]interface{}{
		"name":     userName,
		"password": "wrong_password",
	}
	resp, err = client.Post("/authenticate_user", authPayload)
	if err != nil {
		t.Fatalf("POST /authenticate_user: %v", err)
	}
	// goiardi returns 401 for wrong passwords
	pedant.AssertStatus(t, resp, 401)
}

func TestAuthenticateUserNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	authPayload := map[string]interface{}{
		"name":     "nonexistent_user",
		"password": "anything",
	}
	resp, err := client.Post("/authenticate_user", authPayload)
	if err != nil {
		t.Fatalf("POST /authenticate_user: %v", err)
	}
	// goiardi returns 401 for nonexistent users
	pedant.AssertStatus(t, resp, 401)
}

func TestPrincipalsLookup(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/principals/pivotal")
	if err != nil {
		t.Fatalf("GET /principals/pivotal: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["name"] != "pivotal" {
		t.Errorf("expected name 'pivotal', got %v", body["name"])
	}
}

func TestPrincipalsNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.AdminUser)
	resp, err := client.GetOrg("/principals/nonexistent")
	if err != nil {
		t.Fatalf("GET /principals/nonexistent: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestUsersList(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	resp, err := client.Get("/users")
	if err != nil {
		t.Fatalf("GET /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if _, ok := body["pivotal"]; !ok {
		t.Errorf("expected 'pivotal' in user list, got: %v", body)
	}
}

func TestUsersCreateAndRead(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("test_user")
	u := pedant.NewUser(userName)
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	createBody := pedant.GetJSONBody(t, resp)
	if createBody["uri"] == "" {
		t.Errorf("expected non-empty uri in create response, got %v", createBody["uri"])
	}

	resp, err = client.Get("/users/" + userName)
	if err != nil {
		t.Fatalf("GET /users/%s: %v", userName, err)
	}
	pedant.AssertStatus(t, resp, 200)
	body := pedant.GetJSONBody(t, resp)
	if body["username"] != userName {
		t.Errorf("expected username %q, got %q", userName, body["username"])
	}
}

func TestUsersCreateDuplicate(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("dup_user")
	u := pedant.NewUser(userName)
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

func TestUsersDelete(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("del_user")
	u := pedant.NewUser(userName)

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

func TestUsersNotFound(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	resp, err := client.Get("/users/nonexistent_user")
	if err != nil {
		t.Fatalf("GET /users/nonexistent_user: %v", err)
	}
	pedant.AssertStatus(t, resp, 404)
}

func TestUsersUpdate(t *testing.T) {
	client := testServer.NewClient(testServer.Superuser)
	userName := pedant.UniqueName("upd_user")
	u := pedant.NewUser(userName)
	defer client.Delete("/users/" + userName)

	resp, err := client.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	u["display_name"] = "Updated Display Name"
	resp, err = client.Put("/users/"+userName, u)
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
	if body["display_name"] != "Updated Display Name" {
		t.Errorf("expected display_name 'Updated Display Name', got %v", body["display_name"])
	}
}
