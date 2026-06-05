package main

import (
	"fmt"
	"path/filepath"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/connectors/obsidian"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a sync backend",
	Long: `Connects VaultSync to external backends for syncing your notes.

Supported backends:
  obsidian   Connect to an Obsidian vault folder

Examples:
  vault connect obsidian --path ~/Documents/Obsidian/MyVault
  vault connect obsidian --path ~/Documents/Obsidian/MyVault --subfolder "My Notes"`,
}

var connectObsidianCmd = &cobra.Command{
	Use:   "obsidian",
	Short: "Configure Obsidian sync",
	Long: `Sets up syncing to an Obsidian vault.

VaultSync will copy note files into the Obsidian vault folder under
the configured subfolder (default: "VaultSync").

Example:
  vault connect obsidian --path ~/Documents/Obsidian/MyVault`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		obsidianPath, _ := cmd.Flags().GetString("path")
		subfolder, _ := cmd.Flags().GetString("subfolder")

		if obsidianPath == "" {
			return fmt.Errorf("--path is required")
		}

		absPath, err := filepath.Abs(obsidianPath)
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		cfg.Backends.Obsidian.Enabled = true
		cfg.Backends.Obsidian.VaultPath = absPath
		if subfolder != "" {
			cfg.Backends.Obsidian.Subfolder = subfolder
		}

		conn := obsidian.NewConnector(absPath, cfg.Backends.Obsidian.Subfolder, "", cfg.Backends.Obsidian.Wikilinks)
		if err := conn.Connect(); err != nil {
			return fmt.Errorf("setup obsidian directory: %w", err)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("✓ Connected to Obsidian vault: %s\n", absPath)
		fmt.Printf("  Notes will be synced to: %s/%s\n", absPath, cfg.Backends.Obsidian.Subfolder)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.AddCommand(connectObsidianCmd)

	connectObsidianCmd.Flags().StringP("path", "p", "", "Path to your Obsidian vault (required)")
	connectObsidianCmd.Flags().String("subfolder", "VaultSync", "Subfolder within the Obsidian vault")
}
