package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/ctdk/goiardi/pedant"
	"github.com/ctdk/goiardi/user"
)

// createUserClient creates a new user, generates an RSA key pair for it,
// updates the user's public key, and returns a signing client.
func createUserClient(t *testing.T, adminClient *pedant.ChefSigningClient, name string) *pedant.ChefSigningClient {
	t.Helper()

	u := pedant.NewUser(name)
	resp, err := adminClient.Post("/users", u)
	if err != nil {
		t.Fatalf("POST /users: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	pubDer, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pubKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDer,
	}))

	chefUser, err := user.Get(name)
	if err != nil {
		t.Fatalf("user %s not found: %v", name, err)
	}
	if serr := chefUser.SetPublicKey(pubKeyPEM); serr != nil {
		t.Fatalf("failed to set public key for user %s: %v", name, serr)
	}
	chefUser.Save()

	req := &pedant.TestRequestor{
		Name:       name,
		IsUser:     true,
		PrivateKey: privKey,
	}
	return testServer.NewClient(req)
}

func TestAssociationsCreateAndAccept(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("assoc_user")
	defer adminClient.Delete("/users/" + userName)

	userClient := createUserClient(t, adminClient, userName)

	// Create association request as admin
	reqPayload := map[string]interface{}{"user": userName}
	resp, err := adminClient.PostOrg("/association_requests", reqPayload)
	if err != nil {
		t.Fatalf("POST /association_requests: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)
	body := pedant.GetJSONBody(t, resp)
	if body["uri"] == "" {
		t.Errorf("expected non-empty association request uri, got %v", body["uri"])
	}

	// Accept the association request as the user
	resp, err = userClient.Put("/users/"+userName+"/association_requests/"+userName+"-default", map[string]interface{}{"response": "accept"})
	if err != nil {
		t.Fatalf("PUT association request accept: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}

func TestAssociationsCreateAsNormalUserForbidden(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("assoc_no_perm")
	defer adminClient.Delete("/users/" + userName)

	createUserClient(t, adminClient, userName)

	// Normal user cannot create association requests
	normalClient := testServer.NewClient(testServer.NormalUser)
	reqPayload := map[string]interface{}{"user": userName}
	resp, err := normalClient.PostOrg("/association_requests", reqPayload)
	if err != nil {
		t.Fatalf("POST /association_requests: %v", err)
	}
	pedant.AssertStatus(t, resp, 403)
}

func TestAssociationsReject(t *testing.T) {
	adminClient := testServer.NewClient(testServer.AdminUser)
	userName := pedant.UniqueName("assoc_reject")
	defer adminClient.Delete("/users/" + userName)

	userClient := createUserClient(t, adminClient, userName)

	reqPayload := map[string]interface{}{"user": userName}
	resp, err := adminClient.PostOrg("/association_requests", reqPayload)
	if err != nil {
		t.Fatalf("POST /association_requests: %v", err)
	}
	pedant.AssertStatus(t, resp, 201)

	// Reject the request
	resp, err = userClient.Put("/users/"+userName+"/association_requests/"+userName+"-default", map[string]interface{}{"response": "reject"})
	if err != nil {
		t.Fatalf("PUT association request reject: %v", err)
	}
	pedant.AssertStatus(t, resp, 200)
}
