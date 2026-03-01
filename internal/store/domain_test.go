package store

import (
	"testing"
	"time"
)

func TestDomainAdd_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)

	err := DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	if err != nil {
		t.Fatalf("DomainAdd failed: %v", err)
	}

	// Verify it was created
	domain, err := GetDomainByName(db, "example.com")
	if err != nil {
		t.Fatalf("GetDomainByName failed: %v", err)
	}

	if domain.Domain != "example.com" {
		t.Errorf("expected domain 'example.com', got '%s'", domain.Domain)
	}
	if domain.Provider != "hostinger" {
		t.Errorf("expected provider 'hostinger', got '%s'", domain.Provider)
	}
	if domain.CreatedBy != "testuser" {
		t.Errorf("expected created_by 'testuser', got '%s'", domain.CreatedBy)
	}
	if domain.UpdatedBy != "testuser" {
		t.Errorf("expected updated_by 'testuser', got '%s'", domain.UpdatedBy)
	}
}

func TestDomainAdd_DuplicateFails(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)

	err := DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	if err != nil {
		t.Fatalf("first DomainAdd failed: %v", err)
	}

	err = DomainAdd(db, "example.com", "namecheap", expiry, "testuser")
	if err == nil {
		t.Fatal("expected error when adding duplicate domain, got nil")
	}
}

func TestDomainUpdate_Expiry(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	newExpiry := time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)
	err := DomainUpdate(db, "example.com", nil, &newExpiry, "admin")
	if err != nil {
		t.Fatalf("DomainUpdate failed: %v", err)
	}

	domain, _ := GetDomainByName(db, "example.com")
	if !domain.Expiry.Truncate(24 * time.Hour).Equal(newExpiry) {
		t.Errorf("expected expiry %v, got %v", newExpiry, domain.Expiry)
	}
	if domain.UpdatedBy != "admin" {
		t.Errorf("expected updated_by 'admin', got '%s'", domain.UpdatedBy)
	}
}

func TestDomainUpdate_Provider(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	newProvider := "cloudflare"
	err := DomainUpdate(db, "example.com", &newProvider, nil, "admin")
	if err != nil {
		t.Fatalf("DomainUpdate failed: %v", err)
	}

	domain, _ := GetDomainByName(db, "example.com")
	if domain.Provider != "cloudflare" {
		t.Errorf("expected provider 'cloudflare', got '%s'", domain.Provider)
	}
}

func TestDomainUpdate_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	newProvider := "cloudflare"
	err := DomainUpdate(db, "nonexistent.com", &newProvider, nil, "admin")
	if err == nil {
		t.Fatal("expected error when updating nonexistent domain, got nil")
	}
}

func TestDomainRemove_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	err := DomainRemove(db, "example.com", "admin")
	if err != nil {
		t.Fatalf("DomainRemove failed: %v", err)
	}

	_, err = GetDomainByName(db, "example.com")
	if err == nil {
		t.Fatal("expected error when fetching removed domain, got nil")
	}
}

func TestDomainRemove_BlockedByDNSRecords(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")
	DNSAdd(db, "app.example.com", "localhost:8080", "testuser")

	err := DomainRemove(db, "example.com", "admin")
	if err == nil {
		t.Fatal("expected error when removing domain with DNS records, got nil")
	}
}

func TestDomainRemove_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := DomainRemove(db, "nonexistent.com", "admin")
	if err == nil {
		t.Fatal("expected error when removing nonexistent domain, got nil")
	}
}

func TestDomainSearchAll_NoFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "acrux.ltd", "hostinger", expiry, "testuser")
	DomainAdd(db, "zentix.cloud", "namecheap", expiry, "testuser")

	domains, err := DomainSearchAll(db, "")
	if err != nil {
		t.Fatalf("DomainSearchAll failed: %v", err)
	}

	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(domains))
	}
}

func TestDomainSearchAll_WithFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "acrux.ltd", "hostinger", expiry, "testuser")
	DomainAdd(db, "zentix.cloud", "namecheap", expiry, "testuser")

	domains, err := DomainSearchAll(db, "acrux")
	if err != nil {
		t.Fatalf("DomainSearchAll failed: %v", err)
	}

	if len(domains) != 1 {
		t.Errorf("expected 1 domain matching 'acrux', got %d", len(domains))
	}
	if len(domains) > 0 && domains[0].Domain != "acrux.ltd" {
		t.Errorf("expected 'acrux.ltd', got '%s'", domains[0].Domain)
	}
}

func TestFindParentDomain_ExactMatch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	domain, err := FindParentDomain(db, "example.com")
	if err != nil {
		t.Fatalf("FindParentDomain failed: %v", err)
	}
	if domain.Domain != "example.com" {
		t.Errorf("expected 'example.com', got '%s'", domain.Domain)
	}
}

func TestFindParentDomain_Subdomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	domain, err := FindParentDomain(db, "app.example.com")
	if err != nil {
		t.Fatalf("FindParentDomain failed: %v", err)
	}
	if domain.Domain != "example.com" {
		t.Errorf("expected 'example.com', got '%s'", domain.Domain)
	}
}

func TestFindParentDomain_DeepSubdomain(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	expiry := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	DomainAdd(db, "example.com", "hostinger", expiry, "testuser")

	domain, err := FindParentDomain(db, "api.v2.staging.example.com")
	if err != nil {
		t.Fatalf("FindParentDomain failed: %v", err)
	}
	if domain.Domain != "example.com" {
		t.Errorf("expected 'example.com', got '%s'", domain.Domain)
	}
}

func TestFindParentDomain_NotRegistered(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := FindParentDomain(db, "app.unknown.com")
	if err == nil {
		t.Fatal("expected error for unregistered parent domain, got nil")
	}
}

func TestIsDomainExpired_NotExpired(t *testing.T) {
	d := &Domain{
		Expiry: time.Now().AddDate(1, 0, 0), // 1 year from now
	}
	if IsDomainExpired(d) {
		t.Error("domain should not be expired")
	}
}

func TestIsDomainExpired_Expired(t *testing.T) {
	d := &Domain{
		Expiry: time.Now().AddDate(-1, 0, 0), // 1 year ago
	}
	if !IsDomainExpired(d) {
		t.Error("domain should be expired")
	}
}

func TestIsDomainExpired_Today(t *testing.T) {
	d := &Domain{
		Expiry: time.Now().Truncate(24 * time.Hour), // today
	}
	// Today should NOT be expired (expires at end of day)
	if IsDomainExpired(d) {
		t.Error("domain expiring today should not be considered expired")
	}
}
