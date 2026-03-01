package cli

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/kashifsb/nurix/internal/store"
	"github.com/kashifsb/nurix/internal/vault"
	"github.com/spf13/cobra"
)

var (
	cfg *vault.NurixConfig
	db  *sql.DB
)

var rootCmd = &cobra.Command{
	Use:   "nurix",
	Short: "Nurix — A CLI tool to manage Caddy reverse proxy records backed by PostgreSQL",
	Long: `Nurix is a command-line tool that manages your Caddy web server's reverse proxy
configuration through a PostgreSQL database.

Instead of manually editing your Caddyfile, you register domains and DNS records
through nurix. The Caddyfile is auto-generated from the database after every change.

Getting started:

  1. Set configuration (stored securely in OS keyring):
     nurix set config \
       --caddyfile-path='/etc/caddy/Caddyfile' \
       --dbhost='localhost' \
       --dbport='5432' \
       --dbuser='postgres' \
       --dbpassword='yourpassword' \
       --dbname='nurix'

  2. Run database migrations:
     nurix run db-migration

  3. Register a domain:
     nurix domain add --domain example.com --provider hostinger --expiry 2027-01-01

  4. Add a DNS record:
     nurix dns add --owner app.example.com --target localhost:8080

Use "nurix <command> --help" for more information about any command.`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// requireConfig loads config from OS keyring and connects to the database.
func requireConfig() {
	var err error
	cfg, err = vault.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	db, err = store.Connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// version
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetVersionTemplate("nurix v{{.Version}}\n")

	// update
	rootCmd.AddCommand(updateCmd)

	// --- set ---
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set nurix configuration values",
		Long: `Store configuration values securely in your OS keyring.

Use "nurix set config" to configure database credentials and the Caddyfile path.`,
	}
	setCmd.AddCommand(setConfigCmd)
	rootCmd.AddCommand(setCmd)

	// --- run ---
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run operational commands",
		Long: `Run operational tasks like database migrations.

Use "nurix run db-migration" to initialize or update the database schema.`,
	}
	runCmd.AddCommand(runDBMigrationCmd)
	rootCmd.AddCommand(runCmd)

	// --- domain ---
	domainCmd := &cobra.Command{
		Use:   "domain",
		Short: "Manage registered domains",
		Long: `Register, update, search, and remove domains.

A domain must be registered before DNS records can be created under it.
Expired domains cannot have new or updated DNS records.

Commands:
  add      Register a new domain
  update   Update domain details (expiry, provider)
  remove   Remove a domain (must have no DNS records)
  search   Search/list domains

Examples:
  nurix domain add --domain example.com --provider hostinger --expiry 2027-01-01
  nurix domain search all
  nurix domain update --domain example.com --expiry 2029-01-01
  nurix domain remove --domain example.com`,
	}

	domainSearchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search registered domains",
		Long:  `Use "nurix domain search all" to list all registered domains.`,
	}
	domainSearchCmd.AddCommand(domainSearchAllCmd)

	domainCmd.AddCommand(domainAddCmd)
	domainCmd.AddCommand(domainUpdateCmd)
	domainCmd.AddCommand(domainRemoveCmd)
	domainCmd.AddCommand(domainSearchCmd)
	rootCmd.AddCommand(domainCmd)

	// --- dns ---
	dnsCmd := &cobra.Command{
		Use:   "dns",
		Short: "Manage DNS (reverse proxy) records",
		Long: `Add, update, search, and remove reverse proxy records.

Each record maps a domain/subdomain (owner) to an upstream target.
The parent domain must be registered and not expired.
The Caddyfile is regenerated and Caddy reloaded after every change.

Commands:
  add      Add a new DNS record
  update   Update the target of a DNS record
  remove   Remove a DNS record
  search   Search/list DNS records

Examples:
  nurix dns add --owner app.example.com --target localhost:8080
  nurix dns search all
  nurix dns search all --owner example.com
  nurix dns update --owner app.example.com --target localhost:8443
  nurix dns remove --owner app.example.com`,
	}

	dnsSearchCmd := &cobra.Command{
		Use:   "search",
		Short: "Search DNS records",
		Long:  `Use "nurix dns search all" to list all DNS records.`,
	}
	dnsSearchCmd.AddCommand(dnsSearchAllCmd)

	dnsCmd.AddCommand(dnsAddCmd)
	dnsCmd.AddCommand(dnsUpdateCmd)
	dnsCmd.AddCommand(dnsRemoveCmd)
	dnsCmd.AddCommand(dnsSearchCmd)
	rootCmd.AddCommand(dnsCmd)
}
