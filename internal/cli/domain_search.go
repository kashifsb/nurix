package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kashifsb/nurix/internal/store"
	"github.com/spf13/cobra"
)

var domainSearchAllCmd = &cobra.Command{
	Use:   "all",
	Short: "List all registered domains (optionally filtered)",
	Long: `Show a table of all registered domains.

Examples:
  nurix domain search all
  nurix domain search all --domain example`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		domainFilter, _ := cmd.Flags().GetString("domain")

		domains, err := store.DomainSearchAll(db, domainFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		if len(domains) == 0 {
			fmt.Println("No domains found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tDOMAIN\tPROVIDER\tEXPIRY\tSTATUS\tCREATED BY\tUPDATED BY\tCREATED AT\tUPDATED AT")
		fmt.Fprintln(w, "--\t------\t--------\t------\t------\t----------\t----------\t----------\t----------")

		now := time.Now().Truncate(24 * time.Hour)

		for _, d := range domains {
			status := "✅ Active"
			expiry := d.Expiry.Truncate(24 * time.Hour)
			if now.After(expiry) {
				status = "❌ Expired"
			} else {
				daysLeft := int(expiry.Sub(now).Hours() / 24)
				if daysLeft <= 30 {
					status = fmt.Sprintf("⚠️  %d days left", daysLeft)
				}
			}

			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				d.ID,
				d.Domain,
				d.Provider,
				d.Expiry.Format("2006-01-02"),
				status,
				d.CreatedBy,
				d.UpdatedBy,
				d.CreatedAt.Format("2006-01-02 15:04:05"),
				d.UpdatedAt.Format("2006-01-02 15:04:05"),
			)
		}

		w.Flush()
	},
}

func init() {
	domainSearchAllCmd.Flags().String("domain", "", "Filter by domain name")
}
