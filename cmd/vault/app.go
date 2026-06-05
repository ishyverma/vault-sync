package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
)

func newManager() (*core.Manager, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	vaultDir := cfg.Vault.Path
	if vaultDir == "" {
		dir, err := config.VaultDir()
		if err != nil {
			return nil, err
		}
		vaultDir = filepath.Join(dir, "notes")
	} else {
		vaultDir = filepath.Dir(vaultDir)
	}

	store := storage.NewNoteStore(vaultDir)
	if err := store.Init(); err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}
	tmpl := core.NewTemplateEngine()
	return core.NewManager(vaultDir, store, tmpl), nil
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

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
