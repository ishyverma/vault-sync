package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "vaultd",
	Short: "VaultSync background daemon",
	Long: `vaultd is the background sync daemon for VaultSync.
It watches for file changes and syncs them to connected backends.

Start it once and it runs silently in the background.`,
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetVersionTemplate("VaultSync Daemon {{.Version}}\n")
}
