package vault

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetCurrentUser_ReturnsNonEmpty(t *testing.T) {
	user := GetCurrentUser()

	if user == "" {
		t.Error("GetCurrentUser should return a non-empty string")
	}

	if user == "unknown" {
		t.Skip("whoami returned unknown — may be running in a restricted environment")
	}

	// Should not contain newlines
	if strings.Contains(user, "\n") {
		t.Errorf("GetCurrentUser should not contain newlines, got '%s'", user)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Skip in CI if keyring is not available
	if runtime.GOOS == "linux" {
		t.Skip("skipping keyring test on Linux CI — Secret Service may not be available")
	}

	original := &NurixConfig{
		CaddyfilePath: "/tmp/test-caddyfile",
		DBHost:        "localhost",
		DBPort:        "5432",
		DBUser:        "testuser",
		DBPassword:    "testpass",
		DBName:        "testdb",
	}

	// Save
	if err := SaveConfig(original); err != nil {
		t.Skipf("skipping — keyring not available: %v", err)
	}

	// Load
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.CaddyfilePath != original.CaddyfilePath {
		t.Errorf("CaddyfilePath: expected '%s', got '%s'", original.CaddyfilePath, loaded.CaddyfilePath)
	}
	if loaded.DBHost != original.DBHost {
		t.Errorf("DBHost: expected '%s', got '%s'", original.DBHost, loaded.DBHost)
	}
	if loaded.DBPort != original.DBPort {
		t.Errorf("DBPort: expected '%s', got '%s'", original.DBPort, loaded.DBPort)
	}
	if loaded.DBUser != original.DBUser {
		t.Errorf("DBUser: expected '%s', got '%s'", original.DBUser, loaded.DBUser)
	}
	if loaded.DBPassword != original.DBPassword {
		t.Errorf("DBPassword: expected '%s', got '%s'", original.DBPassword, loaded.DBPassword)
	}
	if loaded.DBName != original.DBName {
		t.Errorf("DBName: expected '%s', got '%s'", original.DBName, loaded.DBName)
	}

	// Cleanup
	DeleteConfig()
}

func TestLoadConfig_NotConfigured(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("skipping keyring test on Linux CI")
	}

	// Delete any existing config
	DeleteConfig()

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when config not set, got nil")
	}

	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error should mention 'not configured', got: %s", err.Error())
	}
}
