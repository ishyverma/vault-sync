package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/connectors/notion"
	"github.com/ishyverma/vault-sync/internal/connectors/obsidian"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/ishyverma/vault-sync/internal/sync"
)

func newManager() (*core.Manager, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	vaultDir := resolveVaultDir(cfg)

	store := storage.NewNoteStore(vaultDir)
	if err := store.Init(); err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}
	tmpl := core.NewTemplateEngine()
	return core.NewManager(vaultDir, store, tmpl), nil
}

func newSyncEngine() (*sync.Engine, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	vaultDir := resolveVaultDir(cfg)
	notesDir := filepath.Join(vaultDir, "notes")

	store := storage.NewNoteStore(vaultDir)
	if err := store.Init(); err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}

	engine := sync.NewEngine(store, notesDir)
	engine.SetRetryLimit(cfg.Sync.QueueRetryLimit)

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

	return engine, nil
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

func openInEditor(path string) error {
	cfg, err := config.Load()
	if err != nil {
		editor := detectEditor()
		return runEditor(editor, path)
	}

	editor := cfg.Vault.Editor
	if editor == "" {
		editor = detectEditor()
	}
	return runEditor(editor, path)
}

var runEditor = func(editor, path string) error {
	return realRunEditor(editor, path)
}

func realRunEditor(editor, path string) error {
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
