package main

import (
	"fmt"
	"path/filepath"

	"github.com/ishyverma/vault-sync/internal/config"
	gitconnector "github.com/ishyverma/vault-sync/internal/connectors/git"
	"github.com/ishyverma/vault-sync/internal/connectors/notion"
	"github.com/ishyverma/vault-sync/internal/connectors/obsidian"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/ishyverma/vault-sync/internal/sync"
	"github.com/ishyverma/vault-sync/internal/tui"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the terminal dashboard",
	Long: `Opens the VaultSync TUI dashboard with:
  - Dashboard overview (stats, recent notes, sync status)
  - Note browser with preview
  - Full-text search
  - Sync monitor
  - Conflict resolver
  - Settings viewer

Use number keys 1-6 or Tab to switch views.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func runTUI() error {
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
	engine.SetRetryLimit(cfg.Sync.QueueRetryLimit)
	engine.SetHooks(cfg.Hooks.PreSync, cfg.Hooks.PostSync, cfg.Hooks.OnConflict)
	if cfg.Backends.Obsidian.Enabled && cfg.Backends.Obsidian.VaultPath != "" {
		obs := obsidian.NewConnector(
			cfg.Backends.Obsidian.VaultPath,
			cfg.Backends.Obsidian.Subfolder,
			notesDir,
			cfg.Backends.Obsidian.Wikilinks,
		)
		engine.RegisterConnector("obsidian", obs)
	}
	if cfg.Backends.Notion.Enabled && cfg.Backends.Notion.Token != "" {
		conn := notion.NewConnector(
			cfg.Backends.Notion.Token,
			cfg.Backends.Notion.TargetPageID,
			cfg.Backends.Notion.DatabaseID,
			notesDir,
		)
		engine.RegisterConnector("notion", conn)
	}

	if cfg.Backends.Git.Enabled && cfg.Backends.Git.RepoPath != "" {
		gc := gitconnector.NewConnector(
			cfg.Backends.Git.RepoPath,
			cfg.Backends.Git.CommitMessage,
			cfg.Backends.Git.Remote,
		)
		engine.RegisterConnector("git", gc)
	}

	tmpl := core.NewTemplateEngine()
	mgr := core.NewManager(vaultDir, store, tmpl)

	return tui.Run(store, engine, cfg, mgr)
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
