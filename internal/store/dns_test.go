package store

import (
	"testing"
	"time"
)

func seedDomain(t *testing.T, db interface {
	Exec(string, ...interface{}) (interface{}, error)
}) {
	// Helper not needed — we use DomainAdd directly
}

func TestDNSAdd_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	err := DNSAdd(db, "app.example.com", "localhost:8080", "testuser")
	if err != nil {
		t.Fatalf("DNSAdd failed: %v", err)
	}

	records, err := DNSSearchAll(db, "app.example.com", "")
	if err != nil {
		t.Fatalf("DNSSearchAll failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	r := records[0]
	if r.Owner != "app.example.com" {
		t.Errorf("expected owner 'app.example.com', got '%s'", r.Owner)
	}
	if r.Target != "localhost:8080" {
		t.Errorf("expected target 'localhost:8080', got '%s'", r.Target)
	}
	if r.Domain != "example.com" {
		t.Errorf("expected domain 'example.com', got '%s'", r.Domain)
	}
	if r.CreatedBy != "testuser" {
		t.Errorf("expected created_by 'testuser', got '%s'", r.CreatedBy)
	}
}

func TestDNSAdd_UnregisteredDomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := DNSAdd(db, "app.unknown.com", "localhost:8080", "testuser")
	if err == nil {
		t.Fatal("expected error when adding DNS for unregistered domain, got nil")
	}
}

func TestDNSAdd_ExpiredDomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) // expired
	DomainAdd(db, "expired.com", "hostinger", expiry, "testuser")

	err := DNSAdd(db, "app.expired.com", "localhost:8080", "testuser")
	if err == nil {
		t.Fatal("expected error when adding DNS for expired domain, got nil")
	}
}

func TestDNSAdd_DuplicateOwner(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")

	err := DNSAdd(db, "app.example.com", "localhost:9090", "testuser")
	if err == nil {
		t.Fatal("expected error when adding duplicate DNS owner, got nil")
	}
}

func TestDNSAdd_BareDomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	// Adding DNS for the bare domain itself should work
	err := DNSAdd(db, "example.com", "localhost:8080", "testuser")
	if err != nil {
		t.Fatalf("DNSAdd for bare domain failed: %v", err)
	}
}

func TestDNSUpdate_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")

	err := DNSUpdate(db, "app.example.com", "localhost:8443", "admin")
	if err != nil {
		t.Fatalf("DNSUpdate failed: %v", err)
	}

	records, _ := DNSSearchAll(db, "app.example.com", "")
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Target != "localhost:8443" {
		t.Errorf("expected target 'localhost:8443', got '%s'", records[0].Target)
	}
	if records[0].UpdatedBy != "admin" {
		t.Errorf("expected updated_by 'admin', got '%s'", records[0].UpdatedBy)
	}
}

func TestDNSUpdate_ExpiredDomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")

	// Expire the domain
	pastExpiry := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	DomainUpdate(db, "example.com", nil, &pastExpiry, "admin")

	err := DNSUpdate(db, "app.example.com", "localhost:9090", "admin")
	if err == nil {
		t.Fatal("expected error when updating DNS for expired domain, got nil")
	}
}

func TestDNSUpdate_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	err := DNSUpdate(db, "nonexistent.example.com", "localhost:9090", "admin")
	if err == nil {
		t.Fatal("expected error when updating nonexistent DNS record, got nil")
	}
}

func TestDNSRemove_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")

	err := DNSRemove(db, "app.example.com", "admin")
	if err != nil {
		t.Fatalf("DNSRemove failed: %v", err)
	}

	records, _ := DNSSearchAll(db, "app.example.com", "")
	if len(records) != 0 {
		t.Errorf("expected 0 records after removal, got %d", len(records))
	}
}

func TestDNSRemove_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := DNSRemove(db, "nonexistent.example.com", "admin")
	if err == nil {
		t.Fatal("expected error when removing nonexistent DNS record, got nil")
	}
}

func TestDNSRemove_AllowedForExpiredDomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")

	// Expire the domain
	pastExpiry := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	DomainUpdate(db, "example.com", nil, &pastExpiry, "admin")

	// Remove should still work for expired domains
	err := DNSRemove(db, "app.example.com", "admin")
	if err != nil {
		t.Fatalf("DNSRemove should work for expired domains: %v", err)
	}
}

func TestDNSSearchAll_NoFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DomainAdd(db, "other.com", "namecheap", expiry, "testuser")

	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")
	DNSAdd(db, "api.example.com", "localhost:3000", "testuser")
	DNSAdd(db, "other.com", "localhost:9090", "testuser")

	records, err := DNSSearchAll(db, "", "")
	if err != nil {
		t.Fatalf("DNSSearchAll failed: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}
}

func TestDNSSearchAll_FilterByOwner(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DomainAdd(db, "other.com", "namecheap", expiry, "testuser")

	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")
	DNSAdd(db, "api.example.com", "localhost:3000", "testuser")
	DNSAdd(db, "other.com", "localhost:9090", "testuser")

	records, err := DNSSearchAll(db, "example", "")
	if err != nil {
		t.Fatalf("DNSSearchAll failed: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records matching 'example', got %d", len(records))
	}
}

func TestDNSSearchAll_FilterByTarget(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")
	DNSAdd(db, "api.example.com", "localhost:3000", "testuser")

	records, err := DNSSearchAll(db, "", "8080")
	if err != nil {
		t.Fatalf("DNSSearchAll failed: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record matching target '8080', got %d", len(records))
	}
}

func TestGetAllGroupedByDomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "acrux.ltd", "hostinger", expiry, "testuser")
	DomainAdd(db, "zentix.cloud", "namecheap", expiry, "testuser")

	DNSAdd(db, "app.acrux.ltd", "localhost:8080", "testuser")
	DNSAdd(db, "api.acrux.ltd", "localhost:3000", "testuser")
	DNSAdd(db, "zentix.cloud", "localhost:9090", "testuser")

	domains, grouped, err := GetAllGroupedByDomain(db)
	if err != nil {
		t.Fatalf("GetAllGroupedByDomain failed: %v", err)
	}

	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(domains))
	}

	// Find acrux.ltd domain ID
	var acruxID int
	for _, d := range domains {
		if d.Domain == "acrux.ltd" {
			acruxID = d.ID
			break
		}
	}

	acruxRecords := grouped[acruxID]
	if len(acruxRecords) != 2 {
		t.Errorf("expected 2 records under acrux.ltd, got %d", len(acruxRecords))
	}
}
