package cli

import (
	"fmt"
	"os"

	"github.com/kashifsb/nurix/internal/caddy"
	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var dnsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the target of an existing DNS record",
	Long: `Change where a domain/subdomain reverse-proxies to.

The parent domain must not be expired.

Example:
  nurix dns update --owner app.example.com --target localhost:8443`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		owner, _ := cmd.Flags().GetString("owner")
		target, _ := cmd.Flags().GetString("target")

		if owner == "" || target == "" {
			fmt.Fprintln(os.Stderr, "❌ Both --owner and --target are required")
			os.Exit(1)
		}

		user := vault.GetCurrentUser()

		if err := store.DNSUpdate(db, owner, target, user); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ DNS record updated: %s → %s\n", owner, target)

		if err := caddy.SyncCaddyfile(db, cfg.CaddyfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	dnsUpdateCmd.Flags().String("owner", "", "Domain/subdomain to update")
	dnsUpdateCmd.Flags().String("target", "", "New upstream target")
}
