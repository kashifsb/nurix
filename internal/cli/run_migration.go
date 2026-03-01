package cli

import (
	"fmt"
	"os"

	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var runDBMigrationCmd = &cobra.Command{
	Use:   "db-migration",
	Short: "Run database migrations to create or update the schema",
	Long: `Creates or updates the required tables in the configured PostgreSQL database.

Migrations are versioned and tracked in a schema_migrations table.
Only pending migrations are applied. Already-applied ones are skipped.

You must run "nurix set config" before this command.

Example:
  nurix run db-migration`,
	Run: func(cmd *cobra.Command, args []string) {
		loadedCfg, err := vault.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		conn, err := store.Connect(loadedCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		fmt.Println("🔧 Running database migrations...")
		fmt.Println("")

		if err := store.RunMigrations(conn); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Migration failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("")
		fmt.Println("✅ Database is ready!")
		fmt.Println("")
		fmt.Println("Next steps:")
		fmt.Println("  nurix domain add --domain example.com --provider hostinger --expiry 2027-01-01")
		fmt.Println("  nurix dns add --owner app.example.com --target localhost:8080")
	},
}
