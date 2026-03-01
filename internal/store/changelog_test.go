package store

import (
	"testing"
	"time"
)

func TestLogChange_RecordsEntry(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := LogChange(db, "domain", 1, "CREATE", "domain", "", "example.com", "testuser")
	if err != nil {
		t.Fatalf("LogChange failed: %v", err)
	}

	entries, err := GetChangelog(db, "domain", 1)
	if err != nil {
		t.Fatalf("GetChangelog failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 changelog entry, got %d", len(entries))
	}

	e := entries[0]
	if e.EntityType != "domain" {
		t.Errorf("expected entity_type 'domain', got '%s'", e.EntityType)
	}
	if e.EntityID != 1 {
		t.Errorf("expected entity_id 1, got %d", e.EntityID)
	}
	if e.Action != "CREATE" {
		t.Errorf("expected action 'CREATE', got '%s'", e.Action)
	}
	if e.FieldName != "domain" {
		t.Errorf("expected field_name 'domain', got '%s'", e.FieldName)
	}
	if e.OldValue != "" {
		t.Errorf("expected empty old_value, got '%s'", e.OldValue)
	}
	if e.NewValue != "example.com" {
		t.Errorf("expected new_value 'example.com', got '%s'", e.NewValue)
	}
	if e.ChangedBy != "testuser" {
		t.Errorf("expected changed_by 'testuser', got '%s'", e.ChangedBy)
	}
}

func TestLogCreate_RecordsMultipleFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	fields := map[string]string{
		"domain":   "example.com",
		"provider": "hostinger",
		"expiry":   "2027-06-15",
	}

	err := LogCreate(db, "domain", 1, fields, "testuser")
	if err != nil {
		t.Fatalf("LogCreate failed: %v", err)
	}

	entries, err := GetChangelog(db, "domain", 1)
	if err != nil {
		t.Fatalf("GetChangelog failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 changelog entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Action != "CREATE" {
			t.Errorf("expected action 'CREATE', got '%s'", e.Action)
		}
	}
}

func TestLogDelete_RecordsMultipleFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	fields := map[string]string{
		"owner":  "app.example.com",
		"target": "localhost:8080",
	}

	err := LogDelete(db, "dns_record", 5, fields, "admin")
	if err != nil {
		t.Fatalf("LogDelete failed: %v", err)
	}

	entries, err := GetChangelog(db, "dns_record", 5)
	if err != nil {
		t.Fatalf("GetChangelog failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 changelog entries, got %d", len(entries))
	}

	for _, e := range entries {
		if e.Action != "DELETE" {
			t.Errorf("expected action 'DELETE', got '%s'", e.Action)
		}
		if e.NewValue != "" {
			t.Errorf("expected empty new_value for DELETE, got '%s'", e.NewValue)
		}
	}
}

func TestDomainAdd_CreatesChangelogEntries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	// DomainAdd should have logged CREATE entries
	entries, err := GetChangelog(db, "domain", 0)
	if err != nil {
		t.Fatalf("GetChangelog failed: %v", err)
	}

	// Should have entries for domain, provider, expiry
	if len(entries) < 3 {
		t.Errorf("expected at least 3 changelog entries for domain create, got %d", len(entries))
	}
}

func TestDomainUpdate_CreatesChangelogEntries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	newProvider := "cloudflare"
	DomainUpdate(db, "example.com", &newProvider, nil, "admin")

	domain, _ := GetDomainByName(db, "example.com")

	entries, err := GetChangelog(db, "domain", domain.ID)
	if err != nil {
		t.Fatalf("GetChangelog failed: %v", err)
	}

	// Find the UPDATE entry for provider
	found := false
	for _, e := range entries {
		if e.Action == "UPDATE" && e.FieldName == "provider" {
			found = true
			if e.OldValue != "hostinger" {
				t.Errorf("expected old_value 'hostinger', got '%s'", e.OldValue)
			}
			if e.NewValue != "cloudflare" {
				t.Errorf("expected new_value 'cloudflare', got '%s'", e.NewValue)
			}
			if e.ChangedBy != "admin" {
				t.Errorf("expected changed_by 'admin', got '%s'", e.ChangedBy)
			}
		}
	}

	if !found {
		t.Error("expected an UPDATE changelog entry for provider, found none")
	}
}

func TestDNSRemove_CreatesChangelogEntries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")

	DNSRemove(db, "app.example.com", "admin")

	entries, err := GetChangelog(db, "dns_record", 0)
	if err != nil {
		t.Fatalf("GetChangelog failed: %v", err)
	}

	// Should have DELETE entries
	deleteFound := false
	for _, e := range entries {
		if e.Action == "DELETE" {
			deleteFound = true
			if e.ChangedBy != "admin" {
				t.Errorf("expected changed_by 'admin', got '%s'", e.ChangedBy)
			}
		}
	}

	if !deleteFound {
		t.Error("expected DELETE changelog entries, found none")
	}
}

func TestGetChangelog_FilterByEntityType(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	LogChange(db, "domain", 1, "CREATE", "domain", "", "example.com", "user1")
	LogChange(db, "dns_record", 1, "CREATE", "owner", "", "app.example.com", "user1")

	entries, err := GetChangelog(db, "domain", 0)
	if err != nil {
		t.Fatalf("GetChangelog failed: %v", err)
	}

	for _, e := range entries {
		if e.EntityType != "domain" {
			t.Errorf("expected only 'domain' entries, got '%s'", e.EntityType)
		}
	}
}
