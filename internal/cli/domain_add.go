package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/kashifsb/nurix/internal/caddy"
	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var domainAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a new domain",
	Long: `Register a domain so that DNS records can be created under it.

DNS records cannot be created for unregistered or expired domains.

Example:
  nurix domain add --domain example.com --provider hostinger --expiry 2027-06-15`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		domainName, _ := cmd.Flags().GetString("domain")
		provider, _ := cmd.Flags().GetString("provider")
		expiryStr, _ := cmd.Flags().GetString("expiry")

		if domainName == "" || expiryStr == "" {
			fmt.Fprintln(os.Stderr, "❌ --domain and --expiry are required")
			os.Exit(1)
		}

		expiry, err := time.Parse("2006-01-02", expiryStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Invalid date format. Use YYYY-MM-DD (e.g., 2027-06-15): %v\n", err)
			os.Exit(1)
		}

		user := vault.GetCurrentUser()

		if err := store.DomainAdd(db, domainName, provider, expiry, user); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Domain registered: %s (provider: %s, expires: %s)\n", domainName, provider, expiryStr)

		if err := caddy.SyncCaddyfile(db, cfg.CaddyfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	domainAddCmd.Flags().String("domain", "", "Domain name (e.g., example.com)")
	domainAddCmd.Flags().String("provider", "", "Domain registrar (e.g., hostinger, namecheap)")
	domainAddCmd.Flags().String("expiry", "", "Expiry date in YYYY-MM-DD format")
}
