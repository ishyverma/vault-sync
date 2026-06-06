package main

import (
	"fmt"
	"path/filepath"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/connectors/notion"
	"github.com/ishyverma/vault-sync/internal/connectors/obsidian"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a sync backend",
	Long: `Connects VaultSync to external backends for syncing your notes.

Supported backends:
  obsidian   Connect to an Obsidian vault folder
  notion     Connect to a Notion workspace

Examples:
  vault connect obsidian --path ~/Documents/Obsidian/MyVault
  vault connect obsidian --path ~/Documents/Obsidian/MyVault --subfolder "My Notes"
  vault connect notion --token ntn_xxxxx --target-page-id <page-id>`,
}

var connectNotionCmd = &cobra.Command{
	Use:   "notion",
	Short: "Configure Notion sync",
	Long: `Sets up syncing to a Notion workspace.

You need a Notion integration token. Create one at:
  https://www.notion.so/my-integrations

The integration must have access to the target page where notes will be stored.

Example:
  vault connect notion --token ntn_xxxxx --target-page-id <page-id>`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _ := cmd.Flags().GetString("token")
		targetPageID, _ := cmd.Flags().GetString("target-page-id")

		if token == "" {
			return fmt.Errorf("--token is required (get one at https://www.notion.so/my-integrations)")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		conn := notion.NewConnector(token, targetPageID, "", "")
		if err := conn.Connect(); err != nil {
			return fmt.Errorf("verify notion token: %w", err)
		}

		cfg.Backends.Notion.Enabled = true
		cfg.Backends.Notion.Token = token
		if targetPageID != "" {
			cfg.Backends.Notion.TargetPageID = targetPageID
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Println("✓ Connected to Notion")
		if targetPageID != "" {
			fmt.Printf("  Notes will be created under target page: %s\n", targetPageID)
		}
		fmt.Println()
		fmt.Println("  Next step: run 'vault sync' to push your notes to Notion")
		return nil
	},
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
	connectCmd.AddCommand(connectNotionCmd)

	connectObsidianCmd.Flags().StringP("path", "p", "", "Path to your Obsidian vault (required)")
	connectObsidianCmd.Flags().String("subfolder", "VaultSync", "Subfolder within the Obsidian vault")
	connectNotionCmd.Flags().String("token", "", "Notion integration token (required)")
	connectNotionCmd.Flags().String("target-page-id", "", "Parent Notion page ID for notes")
}
