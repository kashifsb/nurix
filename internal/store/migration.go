package store

import (
	"database/sql"
	"fmt"
	"time"
)

// Migration represents a single versioned migration.
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// All migrations in order. Add new ones at the bottom with incrementing version.
var migrations = []Migration{
	{
		Version:     1,
		Description: "create domains table",
		SQL: `
			CREATE TABLE IF NOT EXISTS domains (
				id          SERIAL PRIMARY KEY,
				domain      VARCHAR(255) NOT NULL UNIQUE,
				provider    VARCHAR(255) NOT NULL DEFAULT '',
				expiry      DATE NOT NULL,
				created_by  VARCHAR(255) NOT NULL DEFAULT '',
				updated_by  VARCHAR(255) NOT NULL DEFAULT '',
				created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);`,
	},
	{
		Version:     2,
		Description: "create dns_records table",
		SQL: `
			CREATE TABLE IF NOT EXISTS dns_records (
				id          SERIAL PRIMARY KEY,
				owner       VARCHAR(255) NOT NULL UNIQUE,
				target      VARCHAR(255) NOT NULL,
				domain_id   INTEGER NOT NULL REFERENCES domains(id),
				created_by  VARCHAR(255) NOT NULL DEFAULT '',
				updated_by  VARCHAR(255) NOT NULL DEFAULT '',
				created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);`,
	},
	{
		Version:     3,
		Description: "create changelog table",
		SQL: `
			CREATE TABLE IF NOT EXISTS changelog (
				id          SERIAL PRIMARY KEY,
				entity_type VARCHAR(50)  NOT NULL,
				entity_id   INTEGER      NOT NULL,
				action      VARCHAR(20)  NOT NULL,
				field_name  VARCHAR(255) NOT NULL DEFAULT '',
				old_value   TEXT         NOT NULL DEFAULT '',
				new_value   TEXT         NOT NULL DEFAULT '',
				changed_by  VARCHAR(255) NOT NULL DEFAULT '',
				changed_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
			);`,
	},
}

// ensureMigrationsTable creates the schema_migrations tracking table if it doesn't exist.
func ensureMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     INTEGER PRIMARY KEY,
			description VARCHAR(255) NOT NULL DEFAULT '',
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`
	_, err := db.Exec(query)
	return err
}

// getAppliedVersions returns a set of already-applied migration versions.
func getAppliedVersions(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations ORDER BY version ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, nil
}

// RunMigrations applies all pending migrations and tracks them.
func RunMigrations(db *sql.DB) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	applied, err := getAppliedVersions(db)
	if err != nil {
		return fmt.Errorf("failed to read applied migrations: %w", err)
	}

	pending := 0
	for _, m := range migrations {
		if applied[m.Version] {
			fmt.Printf("  ⏭️  v%d: %s (already applied)\n", m.Version, m.Description)
			continue
		}

		fmt.Printf("  ▶️  v%d: %s ... ", m.Version, m.Description)

		tx, err := db.Begin()
		if err != nil {
			fmt.Println("❌")
			return fmt.Errorf("failed to begin transaction for v%d: %w", m.Version, err)
		}

		if _, err := tx.Exec(m.SQL); err != nil {
			tx.Rollback()
			fmt.Println("❌")
			return fmt.Errorf("migration v%d failed: %w", m.Version, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, description, applied_at) VALUES ($1, $2, $3)`,
			m.Version, m.Description, time.Now(),
		); err != nil {
			tx.Rollback()
			fmt.Println("❌")
			return fmt.Errorf("failed to record migration v%d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			fmt.Println("❌")
			return fmt.Errorf("failed to commit migration v%d: %w", m.Version, err)
		}

		fmt.Println("✅")
		pending++
	}

	if pending == 0 {
		fmt.Println("\n  Database is already up to date.")
	} else {
		fmt.Printf("\n  %d migration(s) applied.\n", pending)
	}

	return nil
}
