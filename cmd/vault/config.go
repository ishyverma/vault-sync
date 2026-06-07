package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open the vault config in your editor",
	Long: `Opens ~/.config/vault/config.toml in your configured editor.

Edit the config file directly, save, and exit. VaultSync will pick up
changes on the next command.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		configPath, err := config.ConfigPath()
		if err != nil {
			return fmt.Errorf("config path: %w", err)
		}

		editor := cfg.Vault.Editor
		if editor == "" {
			editor = os.Getenv("EDITOR")
		}
		if editor == "" {
			editor = "vim"
		}

		ecmd := exec.Command(editor, configPath)
		ecmd.Stdin = os.Stdin
		ecmd.Stdout = os.Stdout
		ecmd.Stderr = os.Stderr

		if err := ecmd.Run(); err != nil {
			return fmt.Errorf("editor: %w", err)
		}

		fmt.Println("✓ Config saved")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
