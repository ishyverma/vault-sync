package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "vault",
	Short: "VaultSync — terminal-first, Vim-powered notes that sync everywhere",
	Long: `VaultSync is a local-first note-taking tool for the terminal.
You write in Vim. You save. It syncs — silently, instantly, everywhere.

  vim my-note.md   →   :w   →   ✓ Synced to Notion + Obsidian

Complete documentation available at https://vaultsync.dev`,
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.SetVersionTemplate("VaultSync {{.Version}}\n")
}
