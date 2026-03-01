package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These variables are set at build time via -ldflags
var (
	Version   = "dev"
	BuildDate = "unknown"
	Commit    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the nurix CLI version",
	Long:  `Display the current version, build date, and commit hash of the nurix CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nurix v%s\n", Version)
		fmt.Printf("  Build date: %s\n", BuildDate)
		fmt.Printf("  Commit:     %s\n", Commit)
	},
}
