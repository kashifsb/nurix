package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func getFallbackConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nurix", "config.enc"), nil
}

func getEncryptionKey() []byte {
	hostname, _ := os.Hostname()
	uid := fmt.Sprintf("%d", os.Getuid())

	material := fmt.Sprintf("%s-%s-%s-%s", hostname, uid, runtime.GOOS, runtime.GOARCH)

	if runtime.GOOS == "linux" {
		if b, err := os.ReadFile("/etc/machine-id"); err == nil {
			material += string(b)
		}
	}
	hash := sha256.Sum256([]byte(material))
	return hash[:]
}

func encrypt(plaintext []byte) ([]byte, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decrypt(ciphertext []byte) ([]byte, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("malformed ciphertext")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// SaveConfig serializes config to JSON and stores it in the OS keyring.
// If the keyring is unavailable (e.g. headless Linux), it falls back to an encrypted file.
func SaveConfig(cfg *NurixConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := keyring.Set(serviceName, configKey, string(data)); err != nil {
		// Fallback to encrypted file storage
		fallbackPath, pathErr := getFallbackConfigPath()
		if pathErr != nil {
			return fmt.Errorf("failed to save config to OS keyring (fallback path error: %v): %w", pathErr, err)
		}

		encryptedData, encErr := encrypt(data)
		if encErr != nil {
			return fmt.Errorf("failed to save config to OS keyring (fallback encryption error: %v): %w", encErr, err)
		}

		if errDir := os.MkdirAll(filepath.Dir(fallbackPath), 0700); errDir != nil {
			return fmt.Errorf("failed to save config to OS keyring (fallback mkdir error: %v): %w", errDir, err)
		}

		if errFile := os.WriteFile(fallbackPath, encryptedData, 0600); errFile != nil {
			return fmt.Errorf("failed to save config to OS keyring (fallback file write error: %v): %w", errFile, err)
		}
	}

	return nil
}

// LoadConfig reads config from the OS keyring and deserializes it.
// It will try to read from the encrypted fallback file if the keyring access fails or is not found.
func LoadConfig() (*NurixConfig, error) {
	secret, err := keyring.Get(serviceName, configKey)
	if err != nil {
		fallbackPath, pathErr := getFallbackConfigPath()
		if pathErr == nil {
			if encryptedData, fileErr := os.ReadFile(fallbackPath); fileErr == nil {
				decryptedData, decErr := decrypt(encryptedData)
				if decErr == nil {
					var cfg NurixConfig
					if jsonErr := json.Unmarshal(decryptedData, &cfg); jsonErr != nil {
						return nil, fmt.Errorf("failed to parse config from encrypted fallback file: %w", jsonErr)
					}
					return &cfg, nil
				}
			} else if !os.IsNotExist(fileErr) {
				return nil, fmt.Errorf("failed to read from fallback config file (%s): %w", fallbackPath, fileErr)
			}
		}

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

	var cfg NurixConfig
	if err := json.Unmarshal([]byte(secret), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config from OS keyring: %w", err)
	}

	return &cfg, nil
}

// DeleteConfig removes config from the OS keyring and the encrypted fallback file.
func DeleteConfig() error {
	krErr := keyring.Delete(serviceName, configKey)

	fallbackPath, pathErr := getFallbackConfigPath()
	var fileErr error
	if pathErr == nil {
		fileErr = os.Remove(fallbackPath)
		if os.IsNotExist(fileErr) {
			fileErr = nil
		}
	}

	if krErr != nil && krErr != keyring.ErrNotFound && fileErr != nil {
		return fmt.Errorf("failed to delete config: keyring error (%v), file error (%v)", krErr, fileErr)
	}
	return nil
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
