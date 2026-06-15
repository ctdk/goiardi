# Running goiardi Tests Against a Database Backend

The integration tests in `pedant_test.go` can run against either the default
in-memory backend or against MySQL or PostgreSQL.

## Quick Reference

```sh
# In-memory (default, no external dependencies)
go test ./...

# MySQL
GOIARDI_TEST_DB=mysql \
  GOIARDI_MYSQL_DBNAME=goiardi_test \
  go test ./...

# PostgreSQL
GOIARDI_TEST_DB=postgresql \
  GOIARDI_POSTGRESQL_DBNAME=goiardi_test \
  go test ./...
```

## Connection Parameters

The test infrastructure reads connection parameters from the same environment
variables that goiardi itself uses (see `config/config.go`):

### MySQL

| Variable | Default |
|---|---|
| `GOIARDI_MYSQL_USERNAME` | (empty) |
| `GOIARDI_MYSQL_PASSWORD` | (empty) |
| `GOIARDI_MYSQL_PROTOCOL` | `tcp` |
| `GOIARDI_MYSQL_ADDRESS` | `127.0.0.1` |
| `GOIARDI_MYSQL_PORT` | `3306` |
| `GOIARDI_MYSQL_DBNAME` | **(required)** |

### PostgreSQL

| Variable | Default |
|---|---|
| `GOIARDI_POSTGRESQL_USERNAME` | (empty) |
| `GOIARDI_POSTGRESQL_PASSWORD` | (empty) |
| `GOIARDI_POSTGRESQL_HOST` | `127.0.0.1` |
| `GOIARDI_POSTGRESQL_PORT` | `5432` |
| `GOIARDI_POSTGRESQL_DBNAME` | **(required)** |
| `GOIARDI_POSTGRESQL_SSL_MODE` | `disable` |

## Architecture

The test infrastructure lives in two files:

- **`pedant/db.go`** — reads env vars, connects to the database, manages
  `BackendType` enum (`BackendInMemory`, `BackendMySQL`, `BackendPostgreSQL`)
- **`pedant_test.go`** — `TestMain` calls `pedant.BackendFromEnv()` and
  `pedant.ConnectTestDB()` before starting the test server; `setupTestDB()`
  creates tables and cleans existing data

When `GOIARDI_TEST_DB` is not set (the default), `TestMain` uses the fast
in-memory backend as before — no external dependencies, no behavioral change.