package cli

import (
	"fmt"
	"os"

	"github.com/kashifsb/nurix/internal/caddy"
	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var domainRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a registered domain",
	Long: `Remove a domain from nurix.

If the domain still has DNS records, removal is blocked.
You must remove all DNS records first.

Example:
  nurix domain remove --domain example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		domainName, _ := cmd.Flags().GetString("domain")

		if domainName == "" {
			fmt.Fprintln(os.Stderr, "❌ --domain is required")
			os.Exit(1)
		}

		user := vault.GetCurrentUser()

		if err := store.DomainRemove(db, domainName, user); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Domain '%s' removed\n", domainName)

		if err := caddy.SyncCaddyfile(db, cfg.CaddyfilePath); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	domainRemoveCmd.Flags().String("domain", "", "Domain to remove")
}
