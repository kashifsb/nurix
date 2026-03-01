package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kashifsb/nurix/internal/store"
	"github.com/spf13/cobra"
)

var dnsSearchAllCmd = &cobra.Command{
	Use:   "all",
	Short: "List all DNS records (optionally filtered)",
	Long: `Show a table of all DNS records, optionally filtered by owner or target.

Examples:
  nurix dns search all
  nurix dns search all --owner example.com
  nurix dns search all --target localhost:8080
  nurix dns search all --owner example.com --target localhost:8080`,
	Run: func(cmd *cobra.Command, args []string) {
		requireConfig()

		owner, _ := cmd.Flags().GetString("owner")
		target, _ := cmd.Flags().GetString("target")

		records, err := store.DNSSearchAll(db, owner, target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		if len(records) == 0 {
			fmt.Println("No DNS records found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tOWNER\tTARGET\tDOMAIN\tCREATED BY\tUPDATED BY\tCREATED AT\tUPDATED AT")
		fmt.Fprintln(w, "--\t-----\t------\t------\t----------\t----------\t----------\t----------")

		for _, r := range records {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				r.ID,
				r.Owner,
				r.Target,
				r.Domain,
				r.CreatedBy,
				r.UpdatedBy,
				r.CreatedAt.Format("2006-01-02 15:04:05"),
				r.UpdatedAt.Format("2006-01-02 15:04:05"),
			)
		}

		w.Flush()
	},
}

func init() {
	dnsSearchAllCmd.Flags().String("owner", "", "Filter by owner (partial match, e.g., 'example' matches 'app.example.com')")
	dnsSearchAllCmd.Flags().String("target", "", "Filter by target (partial match, e.g., '8080' matches 'localhost:8080')")
}
