package vault

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "nurix-cli"
	configKey   = "config"
)

// NurixConfig holds all configuration values stored in the OS keyring.
type NurixConfig struct {
	CaddyfilePath string `json:"caddyfile_path"`
	DBHost        string `json:"db_host"`
	DBPort        string `json:"db_port"`
	DBUser        string `json:"db_user"`
	DBPassword    string `json:"db_password"`
	DBName        string `json:"db_name"`
}

// SaveConfig serializes config to JSON and stores it in the OS keyring.
func SaveConfig(cfg *NurixConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := keyring.Set(serviceName, configKey, string(data)); err != nil {
		return fmt.Errorf("failed to save config to OS keyring: %w", err)
	}

	return nil
}

// LoadConfig reads config from the OS keyring and deserializes it.
func LoadConfig() (*NurixConfig, error) {
	secret, err := keyring.Get(serviceName, configKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, fmt.Errorf(
				"nurix is not configured yet\n\n" +
					"Run:\n" +
					"  nurix set config \\\n" +
					"    --caddyfile-path='/etc/caddy/Caddyfile' \\\n" +
					"    --dbhost='localhost' \\\n" +
					"    --dbport='5432' \\\n" +
					"    --dbuser='postgres' \\\n" +
					"    --dbpassword='yourpassword' \\\n" +
					"    --dbname='nurix'",
			)
		}
		return nil, fmt.Errorf("failed to read config from OS keyring: %w", err)
	}

	var cfg NurixConfig
	if err := json.Unmarshal([]byte(secret), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from OS keyring: %w", err)
	}

	return &cfg, nil
}

// DeleteConfig removes config from the OS keyring.
func DeleteConfig() error {
	return keyring.Delete(serviceName, configKey)
}

// GetCurrentUser returns the OS username (equivalent of `whoami`).
func GetCurrentUser() string {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("whoami")
		out, err := cmd.Output()
		if err != nil {
			return "unknown"
		}
		return strings.TrimSpace(string(out))
	default:
		cmd := exec.Command("whoami")
		out, err := cmd.Output()
		if err != nil {
			return "unknown"
		}
		return strings.TrimSpace(string(out))
	}
}
