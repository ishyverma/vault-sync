package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ishyverma/vault-sync/internal/config"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize VaultSync configuration and directories",
	Long: `Sets up your VaultSync environment:
  - Creates ~/.vault/notes/ for your markdown files
  - Creates ~/.config/vault/config.toml with defaults
  - Detects your preferred editor (nvim, vim, nano)
  - Initializes the note index

Run this once before using any other vault commands.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func detectEditor() string {
	for _, e := range []string{os.Getenv("VISUAL"), os.Getenv("EDITOR"), "nvim", "vim", "nano"} {
		if e == "" {
			continue
		}
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}
	return "nvim"
}

func runInit(cmd *cobra.Command) error {
	cfgDir, err := config.ConfigDir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}

	vaultDir, err := config.VaultDir()
	if err != nil {
		return fmt.Errorf("vault dir: %w", err)
	}

	cfgPath, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("config path: %w", err)
	}

	notesDir := filepath.Join(vaultDir, "notes")
	templatesDir := filepath.Join(vaultDir, "templates")
	attachmentsDir := filepath.Join(vaultDir, "attachments")

	dirs := []string{
		cfgDir,
		vaultDir,
		notesDir,
		templatesDir,
		attachmentsDir,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}

	editor := detectEditor()

	cfg := config.DefaultConfig()
	cfg.Vault.Path = notesDir
	cfg.Vault.Editor = editor
	cfg.Vault.TemplateDir = templatesDir

	if err := config.Save(&cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	store := storage.NewNoteStore(vaultDir)
	store.Init()

	fmt.Println("✓ VaultSync initialized")
	fmt.Println()
	fmt.Printf("  Vault path:   %s\n", notesDir)
	fmt.Printf("  Config:       %s\n", cfgPath)
	fmt.Printf("  Editor:       %s\n", editor)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    vault new my-first-note    Create a note")
	fmt.Println("    vault list                 List all notes")
	fmt.Println()

	notesSample := filepath.Join(notesDir, "welcome.md")
	sampleContent := strings.TrimSpace(fmt.Sprintf(`---
title: Welcome to VaultSync
date: %s
tags: [vaultsync, getting-started]
---

# Welcome to VaultSync

You've successfully initialized your vault!

## Quick Start

Create a new note:
  vault new my-note

Open a note:
  vault open my-note

Search your notes:
  vault search "keyword"

List all notes:
  vault list

## Tips

- Notes are plain markdown files stored in %s
- The YAML frontmatter (between the --- lines) stores metadata
- Tags, dates, and titles are automatically indexed
- Run vault --help for all commands
`, timeNow(), notesDir))

	if err := os.WriteFile(notesSample, []byte(sampleContent), 0o644); err != nil {
		return fmt.Errorf("create welcome note: %w", err)
	}

	fm, body, _ := core.ParseFrontmatter(sampleContent)
	store.CreateNote(&storage.Note{
		ID:        generateID(),
		Filename:  "welcome.md",
		Title:     fm.Title,
		Path:      "welcome.md",
		Tags:      fm.Tags,
		WordCount: core.WordCount(body),
	})

	fmt.Printf("  ✓ Created %s/welcome.md\n", notesDir)

	if err := writeDefaultTemplates(templatesDir); err != nil {
		return fmt.Errorf("write templates: %w", err)
	}
	fmt.Printf("  ✓ Created default templates in %s\n", templatesDir)

	return nil
}

func writeDefaultTemplates(dir string) error {
	templates := map[string]string{
		"blank.md": `---
title: "{{title}}"
date: {{date}}
tags: []
---

`,
		"daily.md": `---
title: "Daily Note - {{date}}"
date: {{date}}
tags: [daily]
---

## Today's Focus

## Tasks

- [ ]

## Notes

`,
		"meeting.md": `---
title: "{{title}}"
date: {{date}}
tags: [meeting]
---

# {{title}}

**Date:** {{date}}
**Attendees:**

## Agenda

1.

## Notes

## Action Items

- [ ]

`,
		"project.md": `---
title: "{{title}}"
date: {{date}}
tags: [project]
---

# {{title}}

## Overview

## Goals

## Tasks

- [ ]

## Resources

`,
	}

	for name, content := range templates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", name, err)
			}
		}
	}
	return nil
}

func timeNow() string {
	if d := os.Getenv("VAULT_TEST_DATE"); d != "" {
		return d
	}
	return timeNowFn().Format("2006-01-02")
}

var timeNowFn = func() time.Time {
	return time.Now()
}
