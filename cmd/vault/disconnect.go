package main

import (
	"fmt"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/spf13/cobra"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect [backend]",
	Short: "Disconnect a sync backend",
	Long: `Disables a connected backend and removes its credentials.
Synced note data on the backend is preserved.

Supported backends: obsidian, notion, git

Examples:
  vault disconnect notion
  vault disconnect obsidian
  vault disconnect git`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		switch backend {
		case "notion":
			if !cfg.Backends.Notion.Enabled {
				return fmt.Errorf("notion is not connected")
			}
			cfg.Backends.Notion.Enabled = false
			cfg.Backends.Notion.Token = ""
			cfg.Backends.Notion.TargetPageID = ""
			cfg.Backends.Notion.DatabaseID = ""
		case "obsidian":
			if !cfg.Backends.Obsidian.Enabled {
				return fmt.Errorf("obsidian is not connected")
			}
			cfg.Backends.Obsidian.Enabled = false
			cfg.Backends.Obsidian.VaultPath = ""
			cfg.Backends.Obsidian.Subfolder = ""
		case "git":
			if !cfg.Backends.Git.Enabled {
				return fmt.Errorf("git is not connected")
			}
			cfg.Backends.Git.Enabled = false
			cfg.Backends.Git.RepoPath = ""
		default:
			return fmt.Errorf("unsupported backend: %s (use: notion, obsidian, git)", backend)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("✓ Disconnected %s\n", backend)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}
