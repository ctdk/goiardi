// Package main — integration test server setup for goiardi, ported from
// chef-pedant.
//
// The actual test cases live in pedant_*_test.go files. This file only
// contains TestMain and shared infrastructure.
//
// To run tests against a database backend, set GOIARDI_TEST_DB and the
// relevant connection environment variables:
//
//	# In-memory (default, no external deps):
//	go test ./...
//
//	# MySQL:
//	GOIARDI_TEST_DB=mysql GOIARDI_MYSQL_DBNAME=goiardi_test go test ./...
//
//	# PostgreSQL:
//	GOIARDI_TEST_DB=postgresql GOIARDI_POSTGRESQL_DBNAME=goiardi_test go test ./...
//
//	# Skip slow/DB-only tests:
//	go test -short ./...
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"
	"net/url"

	"github.com/ctdk/goiardi/association"
	"github.com/ctdk/goiardi/client"
	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goiardi/indexer"
	"github.com/ctdk/goiardi/logger"
	"github.com/ctdk/goiardi/organization"
	"github.com/ctdk/goiardi/pedant"
	"github.com/ctdk/goiardi/user"
)

// testServer is the global test server instance.
var testServer *pedant.TestServer

// testOrg is the default organization used for tests.
var testOrg *organization.Organization

// TestMain sets up the test server once for all tests.
func TestMain(m *testing.M) {
	// Register gob types.
	gobRegister()

	// Configure goiardi for testing.
	config.Config = &config.Conf{
		Hostname:       "localhost",
		ProxyHostname:  "localhost",
		ProxyPort:      0,
		Port:           0,
		UseAuth:        true,
		TimeSlew:       "15m",
		TimeSlewDur:    15 * time.Minute,
		JSONReqMaxSize: 1000000,
		ObjMaxSize:     10485760,
		ConfRoot:       os.TempDir(),
		LogLevel:       "fatal",
		DebugLevel:     5,
	}
	logger.SetLevel(logger.FatalLevel)

	// Detect backend from environment.
	backend, dbParams, err := pedant.BackendFromEnv()
	if err != nil {
		log.Fatalf("Error reading GOIARDI_TEST_DB: %v", err)
	}

	// If a DB backend was requested, connect and set up the test database.
	if backend != pedant.BackendInMemory {
		log.Printf("Using %s backend for tests", backend)
		if err := pedant.ConnectTestDB(backend, dbParams); err != nil {
			log.Fatalf("Failed to connect to %s: %v", backend, err)
		}
		defer pedant.CloseTestDB()

		if err := setupTestDB(); err != nil {
			log.Fatalf("Failed to set up test database: %v", err)
		}
	}

	// Initialize data store.
	datastore.New()

	// Create default organization and initialize indexer. Order matters:
	// in DB mode the org must exist before the indexer; in memory mode the
	// indexer uses a dummy org first.
	if config.UsingDB() {
		testOrg = createDefaultOrg()
		indexer.Initialize(config.Config, testOrg)
	} else {
		indexer.Initialize(config.Config, indexer.DefaultDummyOrg)
		testOrg = createDefaultOrg()
	}

	// Create default actors (pivotal, webui, validator, etc.).
	createDefaultActors(testOrg)

	// Set up router using the same registration as production.
	muxer := createRouter()
	handler := &interceptHandler{router: muxer}

	ts := httptest.NewServer(handler)

	// After the test server is started, update goiardi's idea of the
	// public hostname/port so URLs returned by the API match the test
	// server's actual listener. This fixes /containers and /groups list
	// responses which use config.ServerBaseURL().
	u, _ := url.Parse(ts.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	config.Config.Hostname = host
	config.Config.ProxyHostname = host
	config.Config.Port = port
	config.Config.ProxyPort = port

	testServer = &pedant.TestServer{
		BaseURL: ts.URL,
		Backend: backend,
	}

	// Create test requestors. Default actors already exist; we just align
	// their public keys with freshly generated RSA keys for signing.
	testServer.AdminUser = createTestRequestor(config.SuperuserName, true, true)
	testServer.AdminClient = createTestRequestor(config.DefaultWebui, true, false)
	testServer.ValidatorClient = createTestRequestor(config.DefaultValidator, false, false)
	testServer.OutsideUser = createTestRequestor("outside_user", false, false)

	// Create a normal user and client for tests.
	createNormalTestActor()
	testServer.NormalUser = createTestRequestor("pedant_test_user", false, true)
	testServer.NormalClient = createTestRequestor("pedant_test_client", false, false)
	testServer.Superuser = testServer.AdminUser

	// Run tests.
	code := m.Run()

	// Cleanup.
	ts.Close()

	os.Exit(code)
}

// setupTestDB prepares the database for integration tests. This is a stub
// for now; DB-backed test setup will be ported as needed.
func setupTestDB() error {
	return nil
}

// createNormalTestActor creates a non-admin user and client for use in
// tests, if they do not already exist.
func createNormalTestActor() {
	var u *user.User
	if existing, _ := user.Get("pedant_test_user"); existing == nil {
		nu, _ := user.New("pedant_test_user")
		nu.Admin = false
		nu.GenerateKeys()
		nu.Save()
		u = nu
	} else {
		u = existing
	}
	// Ensure the normal user is associated with the default organization.
	if _, err := association.GetAssoc(u, testOrg); err != nil {
		inviter, _ := user.Get(config.SuperuserName)
		if inviter == nil {
			log.Fatalf("pivotal superuser not found")
		}
		assocReq, aerr := association.SetReq(u, testOrg, inviter)
		if aerr != nil {
			log.Fatalf("failed to create association request: %v", aerr)
		}
		if aerr := assocReq.Accept(); aerr != nil {
			log.Fatalf("failed to accept association request: %v", aerr)
		}
	}
	if c, _ := client.Get(testOrg, "pedant_test_client"); c == nil {
		nc, _ := client.New(testOrg, "pedant_test_client")
		nc.Admin = false
		nc.GenerateKeys()
		nc.Save()
	}
	if c, _ := client.Get(testOrg, "outside_user"); c == nil {
		nc, _ := client.New(testOrg, "outside_user")
		nc.Admin = false
		nc.GenerateKeys()
		nc.Save()
	}
}

// createTestRequestor generates a new RSA key pair for the named actor and
// updates that actor's public key so requests can be signed.
func createTestRequestor(name string, isAdmin bool, isUser bool) *pedant.TestRequestor {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("failed to generate test key: %v", err))
	}

	pub := privKey.PublicKey
	pubDer, err := x509.MarshalPKIXPublicKey(&pub)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal public key: %v", err))
	}
	pubKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDer,
	}))

	if isUser {
		u, err := user.Get(name)
		if err == nil && u != nil {
			if serr := u.SetPublicKey(pubKeyPEM); serr != nil {
				panic(fmt.Sprintf("failed to set public key for user %s: %v", name, serr))
			}
			u.Save()
		}
	} else {
		c, err := client.Get(testOrg, name)
		if err == nil && c != nil {
			if serr := c.SetPublicKey(pubKeyPEM); serr != nil {
				panic(fmt.Sprintf("failed to set public key for client %s: %v", name, serr))
			}
			c.Save()
		}
	}

	return &pedant.TestRequestor{
		Name:       name,
		PrivateKey: privKey,
		IsUser:     isUser,
		IsAdmin:    isAdmin,
	}
}
