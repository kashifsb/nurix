package store

import (
	"database/sql"
	"fmt"

	"github.com/kashifsb/nurix/internal/vault"
	_ "github.com/lib/pq"
)

// Connect establishes a connection to the PostgreSQL database.
func Connect(cfg *vault.NurixConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database at %s:%s — %w", cfg.DBHost, cfg.DBPort, err)
	}

	return conn, nil
}
