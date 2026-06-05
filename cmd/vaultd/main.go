package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/connectors/obsidian"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/ishyverma/vault-sync/internal/sync"
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

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background daemon",
	Long: `Starts the VaultSync daemon in the foreground.
It watches the notes directory for file changes and syncs
to all connected backends (Obsidian, etc.).

Run 'vaultd start &' or set up a service to background it.

Press Ctrl+C to stop.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		interval, _ := cmd.Flags().GetInt("interval")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		vaultDir := resolveVaultDir(cfg)
		notesDir := filepath.Join(vaultDir, "notes")

		store := storage.NewNoteStore(vaultDir)
		if err := store.Init(); err != nil {
			return fmt.Errorf("init store: %w", err)
		}

		engine := sync.NewEngine(store, notesDir)

		if cfg.Backends.Obsidian.Enabled && cfg.Backends.Obsidian.VaultPath != "" {
			obs := obsidian.NewConnector(
				cfg.Backends.Obsidian.VaultPath,
				cfg.Backends.Obsidian.Subfolder,
				notesDir,
				cfg.Backends.Obsidian.Wikilinks,
			)
			engine.RegisterConnector("obsidian", obs)
		}

		pollInterval := time.Duration(interval) * time.Second
		if interval <= 0 {
			pollInterval = time.Duration(cfg.Sync.SyncInterval) * time.Second
		}

		daemon := sync.NewDaemon(engine, notesDir, pollInterval)
		if cfg.Backends.Obsidian.Enabled {
			obsidianDir := filepath.Join(cfg.Backends.Obsidian.VaultPath, cfg.Backends.Obsidian.Subfolder)
			daemon.SetObsidianDir(obsidianDir)
		}
		return daemon.Start()
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		vaultDir := resolveVaultDir(cfg)
		notesDir := filepath.Join(vaultDir, "notes")

		store := storage.NewNoteStore(vaultDir)
		if err := store.Init(); err != nil {
			return fmt.Errorf("init store: %w", err)
		}

		engine := sync.NewEngine(store, notesDir)
		daemon := sync.NewDaemon(engine, notesDir, 60*time.Second)

		if err := daemon.Stop(); err != nil {
			return fmt.Errorf("stop daemon: %w", err)
		}

		fmt.Println("✓ vaultd stopped")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if the daemon is running",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		vaultDir := resolveVaultDir(cfg)
		notesDir := filepath.Join(vaultDir, "notes")

		store := storage.NewNoteStore(vaultDir)
		if err := store.Init(); err != nil {
			return fmt.Errorf("init store: %w", err)
		}

		engine := sync.NewEngine(store, notesDir)
		daemon := sync.NewDaemon(engine, notesDir, 60*time.Second)

		running, pid, err := daemon.Status()
		if err != nil {
			return fmt.Errorf("check status: %w", err)
		}

		if running {
			fmt.Printf("vaultd is running (PID %d)\n", pid)
		} else {
			fmt.Println("vaultd is not running")
		}

		return nil
	},
}

func init() {
	rootCmd.SetVersionTemplate("VaultSync Daemon {{.Version}}\n")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)

	startCmd.Flags().IntP("interval", "i", 0, "Poll interval in seconds (default: from config)")
}

func resolveVaultDir(cfg *config.Config) string {
	if cfg.Vault.Path != "" {
		return expandPath(filepath.Dir(cfg.Vault.Path))
	}
	dir, err := config.VaultDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".vault")
	}
	return dir
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
