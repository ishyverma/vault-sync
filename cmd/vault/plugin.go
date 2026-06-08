package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	vaultsync "github.com/ishyverma/vault-sync"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin [install]",
	Short: "Manage VaultSync editor plugins",
	Long: `Installs the VaultSync plugin for Vim or Neovim.

The plugin adds auto-sync on save, :VaultSyncPush and :VaultSyncStatus
commands, and statusline integration.

Examples:
  vault plugin install              # Auto-detect and install
  vault plugin install --vim        # Install for Vim
  vault plugin install --neovim     # Install for Neovim
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] == "install" {
			return runPluginInstall(cmd, args)
		}
		return cmd.Help()
	},
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	vimMode, _ := cmd.Flags().GetBool("vim")
	neovimMode, _ := cmd.Flags().GetBool("neovim")

	var targetDir string
	switch {
	case vimMode:
		targetDir = filepath.Join(os.Getenv("HOME"), ".vim/pack/vaultsync/start/vaultsync")
	case neovimMode:
		targetDir = filepath.Join(os.Getenv("HOME"), ".local/share/nvim/site/pack/vaultsync/start/vaultsync")
	default:
		nvimDir := filepath.Join(os.Getenv("HOME"), ".config/nvim")
		if _, err := os.Stat(nvimDir); err == nil {
			targetDir = filepath.Join(os.Getenv("HOME"), ".local/share/nvim/site/pack/vaultsync/start/vaultsync")
		} else {
			targetDir = filepath.Join(os.Getenv("HOME"), ".vim/pack/vaultsync/start/vaultsync")
		}
	}

	count, err := copyEmbeddedFS(vaultsync.PluginFS, "vim", targetDir)
	if err != nil {
		return fmt.Errorf("install vim plugin: %w", err)
	}

	if strings.Contains(targetDir, "nvim") {
		luaTarget := filepath.Join(os.Getenv("HOME"), ".local/share/nvim/site/pack/vaultsync/start/vaultsync")
		n, err := copyEmbeddedFS(vaultsync.PluginFS, "lua", luaTarget)
		if err != nil {
			return fmt.Errorf("install lua plugin: %w", err)
		}
		count += n

		pluginLoader := filepath.Join(os.Getenv("HOME"), ".config/nvim/after/plugin/vault.lua")
		os.MkdirAll(filepath.Dir(pluginLoader), 0755)
		os.WriteFile(pluginLoader, []byte(
			`-- Auto-installed by vault plugin install
pcall(function() require('vault').setup() end)
`,
		), 0644)
		fmt.Printf("  ✓ Created %s\n", pluginLoader)
	}

	fmt.Printf("\n  ✓ Installed %d file(s) to %s\n\n", count, targetDir)
	fmt.Println("  Next steps:")
	fmt.Println("    Vim:     The plugin is loaded automatically via packpath")
	fmt.Println("    Neovim:  The plugin is loaded automatically via packpath")
	fmt.Println("    Restart your editor and run :VaultSyncPush to test")
	return nil
}

func copyEmbeddedFS(efs fs.FS, srcDir, targetDir string) (int, error) {
	var count int
	err := fs.WalkDir(efs, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)
		if rel == "." {
			return nil
		}
		target := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(efs, path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
		fmt.Printf("  ✓ %s\n", target)
		count++
		return nil
	})
	return count, err
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.Flags().Bool("vim", false, "Install for Vim")
	pluginCmd.Flags().Bool("neovim", false, "Install for Neovim")
}
