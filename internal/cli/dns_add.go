package cli

import (
	"fmt"
	"os"

	"github.com/kashifsb/nurix/internal/caddy"
	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var dnsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new DNS (reverse proxy) record",
	Long: `Add a reverse proxy record mapping a domain/subdomain to an upstream target.

The parent domain must be registered and not expired.
The Caddyfile is regenerated and Caddy reloaded after adding.

Examples:
  nurix dns add --owner app.example.com --target localhost:8080
  nurix dns add --owner example.com --target localhost:3000`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		owner, _ := cmd.Flags().GetString("owner")
		target, _ := cmd.Flags().GetString("target")

		if owner == "" || target == "" {
			fmt.Fprintln(os.Stderr, "❌ Both --owner and --target are required")
			os.Exit(1)
		}

		user := vault.GetCurrentUser()

		if err := store.DNSAdd(db, owner, target, user); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ DNS record added: %s → %s\n", owner, target)

		if err := caddy.SyncCaddyfile(db, cfg.CaddyfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	dnsAddCmd.Flags().String("owner", "", "Domain/subdomain (e.g., app.example.com)")
	dnsAddCmd.Flags().String("target", "", "Upstream target (e.g., localhost:8080)")
}
