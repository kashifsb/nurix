package cli

import (
	"fmt"
	"os"

	"github.com/kashifsb/nurix/internal/caddy"
	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var dnsRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a DNS record",
	Long: `Remove a reverse proxy record. Works even if the parent domain is expired.

The Caddyfile is regenerated and Caddy reloaded after removal.

Example:
  nurix dns remove --owner app.example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		owner, _ := cmd.Flags().GetString("owner")

		if owner == "" {
			fmt.Fprintln(os.Stderr, "❌ --owner is required")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Usage:")
			fmt.Fprintln(os.Stderr, "  nurix dns remove --owner app.example.com")
			os.Exit(1)
		}

		user := vault.GetCurrentUser()

		if err := store.DNSRemove(db, owner, user); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ DNS record removed: %s\n", owner)

		if err := caddy.SyncCaddyfile(db, cfg.CaddyfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Caddyfile sync failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	dnsRemoveCmd.Flags().String("owner", "", "Domain/subdomain to remove (e.g., app.example.com)")
}
