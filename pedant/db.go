// Package pedant provides integration test helpers for goiardi, including
// support for running tests against a MySQL or PostgreSQL backend.
package pedant

import (
	"fmt"
	"os"
	"strings"

	"github.com/ctdk/goiardi/config"
	"github.com/ctdk/goiardi/datastore"
)

// BackendType describes which backend the test server is using.
type BackendType int

const (
	// BackendInMemory uses the fast in-memory cache backend.
	BackendInMemory BackendType = iota
	// BackendMySQL uses a MySQL database backend.
	BackendMySQL
	// BackendPostgreSQL uses a PostgreSQL database backend.
	BackendPostgreSQL
)

// String returns the human-readable name for the backend type.
func (b BackendType) String() string {
	switch b {
	case BackendMySQL:
		return "mysql"
	case BackendPostgreSQL:
		return "postgresql"
	default:
		return "in-memory"
	}
}

// BackendFromEnv reads GOIARDI_TEST_DB and returns the requested backend type
// and the connection parameters as a config struct.
//
// If GOIARDI_TEST_DB is not set, BackendInMemory is returned.
// If it is set to "mysql" or "postgresql", the connection parameters are read
// from the same env vars that goiardi itself uses (GOIARDI_MYSQL_* and
// GOIARDI_POSTGRESQL_* respectively).
//
// Returns an error if an unknown backend is requested or if required
// connection parameters are missing.
func BackendFromEnv() (BackendType, interface{}, error) {
	dbType := os.Getenv("GOIARDI_TEST_DB")
	if dbType == "" {
		return BackendInMemory, nil, nil
	}

	switch strings.ToLower(dbType) {
	case "mysql":
		cfg := readMySQLConfig()
		if cfg.Dbname == "" {
			return BackendMySQL, nil, fmt.Errorf("GOIARDI_TEST_DB=mysql but GOIARDI_MYSQL_DBNAME is not set")
		}
		return BackendMySQL, cfg, nil
	case "postgresql", "postgres":
		cfg := readPostgreSQLConfig()
		if cfg.Dbname == "" {
			return BackendPostgreSQL, nil, fmt.Errorf("GOIARDI_TEST_DB=postgresql but GOIARDI_POSTGRESQL_DBNAME is not set")
		}
		return BackendPostgreSQL, cfg, nil
	default:
		return BackendInMemory, nil, fmt.Errorf("unknown GOIARDI_TEST_DB value: %q (expected mysql, postgresql, or empty)", dbType)
	}
}

// ConnectTestDB connects to the test database and sets up goiardi's
// datastore.Dbh handle. It also configures config.Config.UseMySQL or
// UsePostgreSQL as appropriate.
func ConnectTestDB(backend BackendType, params interface{}) error {
	var engine string
	switch backend {
	case BackendMySQL:
		engine = "mysql"
		config.Config.UseMySQL = true
	case BackendPostgreSQL:
		engine = "postgresql"
		config.Config.UsePostgreSQL = true
	default:
		return nil
	}

	db, err := datastore.ConnectDB(engine, params)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", engine, err)
	}
	datastore.Dbh = db
	return nil
}

// CloseTestDB cleans up the test database connection, if one was opened.
func CloseTestDB() {
	if datastore.Dbh != nil {
		datastore.Dbh.Close()
		datastore.Dbh = nil
	}
}

// readMySQLConfig reads MySQL connection parameters from environment
// variables.  Uses the same env vars as goiardi itself.
func readMySQLConfig() config.MySQLdb {
	return config.MySQLdb{
		Username: os.Getenv("GOIARDI_MYSQL_USERNAME"),
		Password: os.Getenv("GOIARDI_MYSQL_PASSWORD"),
		Protocol: envOrDefault("GOIARDI_MYSQL_PROTOCOL", "tcp"),
		Address:  envOrDefault("GOIARDI_MYSQL_ADDRESS", "127.0.0.1"),
		Port:     envOrDefault("GOIARDI_MYSQL_PORT", "3306"),
		Dbname:   os.Getenv("GOIARDI_MYSQL_DBNAME"),
	}
}

// readPostgreSQLConfig reads PostgreSQL connection parameters from
// environment variables.  Uses the same env vars as goiardi itself.
func readPostgreSQLConfig() config.PostgreSQLdb {
	return config.PostgreSQLdb{
		Username: os.Getenv("GOIARDI_POSTGRESQL_USERNAME"),
		Password: os.Getenv("GOIARDI_POSTGRESQL_PASSWORD"),
		Host:     envOrDefault("GOIARDI_POSTGRESQL_HOST", "127.0.0.1"),
		Port:     envOrDefault("GOIARDI_POSTGRESQL_PORT", "5432"),
		Dbname:   os.Getenv("GOIARDI_POSTGRESQL_DBNAME"),
		SSLMode:  envOrDefault("GOIARDI_POSTGRESQL_SSL_MODE", "disable"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}