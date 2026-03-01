package store

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// testDBConfig reads DB connection info from environment variables.
// Falls back to defaults suitable for local dev / CI.
func testDBConfig() (string, string, string, string, string) {
	host := envOrDefault("NURIX_TEST_DB_HOST", "localhost")
	port := envOrDefault("NURIX_TEST_DB_PORT", "5432")
	user := envOrDefault("NURIX_TEST_DB_USER", "postgres")
	password := envOrDefault("NURIX_TEST_DB_PASSWORD", "postgres")
	dbname := envOrDefault("NURIX_TEST_DB_NAME", "nurix_test")
	return host, port, user, password, dbname
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// setupTestDB creates a fresh connection and runs all migrations.
// It returns the DB connection and a cleanup function that drops all tables.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	host, port, user, password, dbname := testDBConfig()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("skipping test — cannot connect to test database: %v", err)
	}

	// Clean slate
	dropAllTables(t, db)

	// Run migrations
	if err := RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	cleanup := func() {
		dropAllTables(t, db)
		db.Close()
	}

	return db, cleanup
}

func dropAllTables(t *testing.T, db *sql.DB) {
	t.Helper()

	tables := []string{
		"changelog",
		"dns_records",
		"domains",
		"schema_migrations",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Fatalf("failed to drop table %s: %v", table, err)
		}
	}
}
