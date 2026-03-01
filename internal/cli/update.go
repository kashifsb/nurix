package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	githubOwner = "kashifsb"
	githubRepo  = "nurix"
)

// gitHubRelease represents the relevant fields from the GitHub Releases API.
type gitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []gitHubAsset `json:"assets"`
}

type gitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update nurix to the latest version",
	Long: `Check GitHub for the latest release and replace the current binary.

Your configuration (OS keyring) and database are not affected.
After updating, run "nurix run db-migration" to apply any new schema changes.

Examples:
  nurix update
  nurix update --check`,
	Run: func(cmd *cobra.Command, args []string) {
		checkOnly, _ := cmd.Flags().GetBool("check")

		fmt.Println("🔍 Checking for updates...")
		fmt.Println("")

		release, err := fetchLatestRelease()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to check for updates: %v\n", err)
			os.Exit(1)
		}

		latestVersion := strings.TrimPrefix(release.TagName, "v")
		currentVersion := Version

		fmt.Printf("  Current version: v%s\n", currentVersion)
		fmt.Printf("  Latest version:  v%s\n", latestVersion)
		fmt.Println("")

		if latestVersion == currentVersion {
			fmt.Println("✅ You're already on the latest version!")
			return
		}

		if checkOnly {
			fmt.Printf("📦 A new version is available: v%s → v%s\n", currentVersion, latestVersion)
			fmt.Println("")
			fmt.Println("Run 'nurix update' to install it.")
			return
		}

		// Find the right asset for this OS/arch
		assetName := buildAssetName()
		var downloadURL string
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		if downloadURL == "" {
			fmt.Fprintf(os.Stderr, "❌ No prebuilt binary found for %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Update manually:")
			fmt.Fprintln(os.Stderr, "  git pull origin main && make install")
			os.Exit(1)
		}

		fmt.Printf("⬇️  Downloading v%s ...\n", latestVersion)

		binaryPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to determine current binary path: %v\n", err)
			os.Exit(1)
		}

		// Resolve symlinks
		binaryPath, err = resolveSymlink(binaryPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to resolve binary path: %v\n", err)
			os.Exit(1)
		}

		// Download to a temp file first
		tmpFile, err := os.CreateTemp("", "nurix-update-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create temp file: %v\n", err)
			os.Exit(1)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		resp, err := http.Get(downloadURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Download failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "❌ Download failed with status: %s\n", resp.Status)
			os.Exit(1)
		}

		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to write downloaded binary: %v\n", err)
			os.Exit(1)
		}
		tmpFile.Close()

		// Make executable
		if err := os.Chmod(tmpPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to set permissions: %v\n", err)
			os.Exit(1)
		}

		// Replace the current binary
		if err := replaceBinary(tmpPath, binaryPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to replace binary: %v\n", err)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintf(os.Stderr, "Try manually:\n")
			fmt.Fprintf(os.Stderr, "  sudo cp %s %s\n", tmpPath, binaryPath)
			os.Exit(1)
		}

		fmt.Println("")
		fmt.Printf("✅ Updated successfully: v%s → v%s\n", currentVersion, latestVersion)
		fmt.Println("")
		fmt.Println("Next step — apply any new database migrations:")
		fmt.Println("  nurix run db-migration")
	},
}

func fetchLatestRelease() (*gitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "nurix-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %s", resp.Status)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// buildAssetName returns the expected binary name for the current platform.
// Convention: nurix-{os}-{arch}
// Examples: nurix-linux-amd64, nurix-darwin-arm64, nurix-linux-arm64
func buildAssetName() string {
	return fmt.Sprintf("nurix-%s-%s", runtime.GOOS, runtime.GOARCH)
}

func resolveSymlink(path string) (string, error) {
	resolved, err := os.Readlink(path)
	if err != nil {
		// Not a symlink, return original
		return path, nil
	}
	return resolved, nil
}

func replaceBinary(src, dst string) error {
	// Try direct rename first (works if same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fallback: use sudo cp
	cmd := exec.Command("sudo", "cp", src, dst)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	updateCmd.Flags().Bool("check", false, "Only check for updates without installing")
}
