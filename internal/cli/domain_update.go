package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var domainUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a registered domain's details",
	Long: `Update the expiry date or provider of a registered domain.

Examples:
  nurix domain update --domain example.com --expiry 2029-01-01
  nurix domain update --domain example.com --provider cloudflare`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		domainName, _ := cmd.Flags().GetString("domain")
		expiryStr, _ := cmd.Flags().GetString("expiry")
		providerStr, _ := cmd.Flags().GetString("provider")

		if domainName == "" {
			fmt.Fprintln(os.Stderr, "❌ --domain is required")
			os.Exit(1)
		}

		var expiryPtr *time.Time
		if expiryStr != "" {
			expiry, err := time.Parse("2006-01-02", expiryStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Invalid date format. Use YYYY-MM-DD: %v\n", err)
				os.Exit(1)
			}
			expiryPtr = &expiry
		}

		var providerPtr *string
		if cmd.Flags().Changed("provider") {
			providerPtr = &providerStr
		}

		if expiryPtr == nil && providerPtr == nil {
			fmt.Fprintln(os.Stderr, "❌ Provide at least --expiry or --provider to update")
			os.Exit(1)
		}

		user := vault.GetCurrentUser()

		if err := store.DomainUpdate(db, domainName, providerPtr, expiryPtr, user); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Domain '%s' updated successfully\n", domainName)
	},
}

func init() {
	domainUpdateCmd.Flags().String("domain", "", "Domain name to update")
	domainUpdateCmd.Flags().String("expiry", "", "New expiry date (YYYY-MM-DD)")
	domainUpdateCmd.Flags().String("provider", "", "New provider name")
}
