package store

import (
	"testing"
)

func TestRunMigrations_CreatesAllTables(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify all tables exist by querying them
	tables := []string{"domains", "dns_records", "changelog", "schema_migrations"}

	for _, table := range tables {
		var exists bool
		err := db.QueryRow(
			`SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)`, table,
		).Scan(&exists)

		if err != nil {
			t.Fatalf("failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s should exist after migration", table)
		}
	}
}

func TestRunMigrations_TracksVersions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// All migration versions should be recorded
	rows, err := db.Query(`SELECT version, description FROM schema_migrations ORDER BY version ASC`)
	if err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		var description string
		if err := rows.Scan(&version, &description); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		versions = append(versions, version)

		if description == "" {
			t.Errorf("migration v%d has empty description", version)
		}
	}

	if len(versions) != len(migrations) {
		t.Errorf("expected %d migrations recorded, got %d", len(migrations), len(versions))
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Run migrations again — should not fail
	if err := RunMigrations(db); err != nil {
		t.Fatalf("second migration run should not fail: %v", err)
	}

	// Should still have the same number of versions
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}

	if count != len(migrations) {
		t.Errorf("expected %d migrations after idempotent run, got %d", len(migrations), count)
	}
}
