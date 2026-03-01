package cli

import (
	"fmt"
	"os"

	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var setConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Set all nurix configuration values",
	Long: `Configure nurix with your database credentials and Caddyfile path.
All values are stored securely in your OS keyring (macOS Keychain,
Linux Secret Service, Windows Credential Manager). Nothing is written to disk.

This must be run before any other nurix command.

Example:
  nurix set config \
    --caddyfile-path='/etc/caddy/Caddyfile' \
    --dbhost='localhost' \
    --dbport='5432' \
    --dbuser='postgres' \
    --dbpassword='mysecretpassword' \
    --dbname='nurix'`,
	Run: func(cmd *cobra.Command, args []string) {
		caddyfilePath, _ := cmd.Flags().GetString("caddyfile-path")
		dbhost, _ := cmd.Flags().GetString("dbhost")
		dbport, _ := cmd.Flags().GetString("dbport")
		dbuser, _ := cmd.Flags().GetString("dbuser")
		dbpassword, _ := cmd.Flags().GetString("dbpassword")
		dbname, _ := cmd.Flags().GetString("dbname")

		if caddyfilePath == "" || dbhost == "" || dbport == "" || dbuser == "" || dbpassword == "" || dbname == "" {
			fmt.Fprintln(os.Stderr, "❌ All flags are required:")
			fmt.Fprintln(os.Stderr, "   --caddyfile-path, --dbhost, --dbport, --dbuser, --dbpassword, --dbname")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Example:")
			fmt.Fprintln(os.Stderr, "  nurix set config \\")
			fmt.Fprintln(os.Stderr, "    --caddyfile-path='/etc/caddy/Caddyfile' \\")
			fmt.Fprintln(os.Stderr, "    --dbhost='localhost' \\")
			fmt.Fprintln(os.Stderr, "    --dbport='5432' \\")
			fmt.Fprintln(os.Stderr, "    --dbuser='postgres' \\")
			fmt.Fprintln(os.Stderr, "    --dbpassword='mypassword' \\")
			fmt.Fprintln(os.Stderr, "    --dbname='nurix'")
			os.Exit(1)
		}

		newCfg := &vault.NurixConfig{
			CaddyfilePath: caddyfilePath,
			DBHost:        dbhost,
			DBPort:        dbport,
			DBUser:        dbuser,
			DBPassword:    dbpassword,
			DBName:        dbname,
		}

		if err := vault.SaveConfig(newCfg); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Configuration saved securely to OS keyring")
		fmt.Println("")
		fmt.Println("Next step — run database migrations:")
		fmt.Println("  nurix run db-migration")
	},
}

func init() {
	setConfigCmd.Flags().String("caddyfile-path", "", "Full path to the Caddyfile (e.g., /etc/caddy/Caddyfile)")
	setConfigCmd.Flags().String("dbhost", "", "PostgreSQL host (e.g., localhost)")
	setConfigCmd.Flags().String("dbport", "", "PostgreSQL port (e.g., 5432)")
	setConfigCmd.Flags().String("dbuser", "", "PostgreSQL username")
	setConfigCmd.Flags().String("dbpassword", "", "PostgreSQL password")
	setConfigCmd.Flags().String("dbname", "", "PostgreSQL database name (e.g., nurix)")
}
