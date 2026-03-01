package store

import (
	"database/sql"
	"fmt"
	"time"
)

type ChangelogEntry struct {
	ID         int
	EntityType string
	EntityID   int
	Action     string
	FieldName  string
	OldValue   string
	NewValue   string
	ChangedBy  string
	ChangedAt  time.Time
}

// LogChange records a single field change in the changelog.
func LogChange(db *sql.DB, entityType string, entityID int, action, fieldName, oldValue, newValue, changedBy string) error {
	query := `
		INSERT INTO changelog (entity_type, entity_id, action, field_name, old_value, new_value, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.Exec(query, entityType, entityID, action, fieldName, oldValue, newValue, changedBy)
	if err != nil {
		return fmt.Errorf("failed to write changelog: %w", err)
	}
	return nil
}

// LogCreate records a CREATE action with all field values.
func LogCreate(db *sql.DB, entityType string, entityID int, fields map[string]string, changedBy string) error {
	for field, value := range fields {
		if err := LogChange(db, entityType, entityID, "CREATE", field, "", value, changedBy); err != nil {
			return err
		}
	}
	return nil
}

// LogDelete records a DELETE action.
func LogDelete(db *sql.DB, entityType string, entityID int, fields map[string]string, changedBy string) error {
	for field, value := range fields {
		if err := LogChange(db, entityType, entityID, "DELETE", field, value, "", changedBy); err != nil {
			return err
		}
	}
	return nil
}

// GetChangelog retrieves changelog entries, optionally filtered.
func GetChangelog(db *sql.DB, entityType string, entityID int) ([]ChangelogEntry, error) {
	query := `SELECT id, entity_type, entity_id, action, field_name, old_value, new_value, changed_by, changed_at
		FROM changelog WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if entityType != "" {
		query += fmt.Sprintf(" AND entity_type = $%d", argIdx)
		args = append(args, entityType)
		argIdx++
	}
	if entityID > 0 {
		query += fmt.Sprintf(" AND entity_id = $%d", argIdx)
		args = append(args, entityID)
		argIdx++
	}

	query += " ORDER BY changed_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ChangelogEntry
	for rows.Next() {
		var e ChangelogEntry
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.Action, &e.FieldName, &e.OldValue, &e.NewValue, &e.ChangedBy, &e.ChangedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
