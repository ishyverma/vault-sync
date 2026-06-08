package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ishyverma/vault-sync/internal/config"
	notionapi "github.com/ishyverma/vault-sync/internal/connectors/notion"
	obsidianapi "github.com/ishyverma/vault-sync/internal/connectors/obsidian"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [backend]",
	Short: "Bulk-import notes from a backend",
	Long: `Bulk-imports existing notes from a connected backend into the local vault.

Supported backends: notion, obsidian

Examples:
  vault import notion              Import all Notion pages
  vault import notion --limit 10
  vault import obsidian            Import notes from connected Obsidian vault`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend := args[0]

		switch backend {
		case "notion":
			return importFromNotion(cmd)
		case "obsidian":
			return importFromObsidian(cmd)
		default:
			return fmt.Errorf("unsupported backend: %s (use: notion, obsidian)", backend)
		}
	},
}

func importFromNotion(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !cfg.Backends.Notion.Enabled || cfg.Backends.Notion.Token == "" {
		return fmt.Errorf("notion backend not configured (run: vault connect notion)")
	}

	mgr, err := newManager()
	if err != nil {
		return fmt.Errorf("open vault: %w", err)
	}

	store, err := newStore()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	notesDir := mgr.NotesDir()

	conn := notionapi.NewConnector(
		cfg.Backends.Notion.Token,
		cfg.Backends.Notion.TargetPageID,
		cfg.Backends.Notion.DatabaseID,
		notesDir,
	)

	if err := conn.Connect(); err != nil {
		return fmt.Errorf("connect to notion: %w", err)
	}

	fmt.Println("Searching for Notion pages...")

	pages, err := conn.Client().Search("")
	if err != nil {
		return fmt.Errorf("search notion: %w", err)
	}

	if len(pages) == 0 {
		fmt.Println("No Notion pages found to import")
		return nil
	}

	limit, _ := cmd.Flags().GetInt("limit")
	if limit > 0 && limit < len(pages) {
		pages = pages[:limit]
	}

	var imported, skipped int
	for _, page := range pages {
		title := extractTitle(page.Properties)
		if title == "" {
			title = "untitled"
		}

		filename := strings.ReplaceAll(strings.ToLower(title), " ", "-") + ".md"
		localPath := filepath.Join(notesDir, filename)

		if existing, err := store.FindNoteByFilename(filename); err == nil && existing != nil {
			fmt.Printf("  ⏭ %s (already exists)\n", filename)
			skipped++
			continue
		}

		content, err := conn.Pull(page.ID)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", filename, err)
			continue
		}

		if err := os.WriteFile(localPath, []byte(content), 0o644); err != nil {
			fmt.Printf("  ✗ %s: write error: %v\n", filename, err)
			continue
		}

		noteID, genErr := core.GenerateID()
		if genErr != nil {
			fmt.Printf("  ✗ %s: generate id error: %v\n", filename, genErr)
			continue
		}
		note := &storage.Note{
			ID:       noteID,
			Filename: filename,
			Title:    title,
			Path:     filename,
		}
		if err := store.CreateNote(note); err != nil {
			fmt.Printf("  ⚠ %s: created file but DB error: %v\n", filename, err)
		}

		store.UpsertSyncState(&storage.SyncState{
			NoteID:   note.ID,
			Backend:  "notion",
			RemoteID: page.ID,
			Status:   "synced",
		})

		fmt.Printf("  ✓ %s\n", filename)
		imported++
	}

	fmt.Printf("\n%d imported, %d skipped\n", imported, skipped)
	return nil
}

func importFromObsidian(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !cfg.Backends.Obsidian.Enabled || cfg.Backends.Obsidian.VaultPath == "" {
		return fmt.Errorf("obsidian backend not configured (run: vault connect obsidian)")
	}

	mgr, err := newManager()
	if err != nil {
		return fmt.Errorf("open vault: %w", err)
	}

	store, err := newStore()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	notesDir := mgr.NotesDir()

	conn := obsidianapi.NewConnector(
		cfg.Backends.Obsidian.VaultPath,
		cfg.Backends.Obsidian.Subfolder,
		notesDir,
		cfg.Backends.Obsidian.Wikilinks,
	)

	if err := conn.Connect(); err != nil {
		return fmt.Errorf("connect to obsidian: %w", err)
	}

	sourceDir := conn.TargetDir()
	fmt.Printf("Scanning %s for markdown files...\n", sourceDir)

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("read obsidian directory: %w", err)
	}

	var imported, skipped int
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
			continue
		}

		filename := f.Name()
		if existing, findErr := store.FindNoteByFilename(filename); findErr == nil && existing != nil {
			fmt.Printf("  ⏭ %s (already exists)\n", filename)
			skipped++
			continue
		}

		srcPath := filepath.Join(sourceDir, filename)
		content, readErr := os.ReadFile(srcPath)
		if readErr != nil {
			fmt.Printf("  ✗ %s: %v\n", filename, readErr)
			continue
		}

		dstPath := filepath.Join(notesDir, filename)
		if err := os.WriteFile(dstPath, content, 0o644); err != nil {
			fmt.Printf("  ✗ %s: write error: %v\n", filename, err)
			continue
		}

		noteID, genErr := core.GenerateID()
		if genErr != nil {
			fmt.Printf("  ✗ %s: generate id error: %v\n", filename, genErr)
			continue
		}

		title := strings.TrimSuffix(filename, ".md")
		note := &storage.Note{
			ID:       noteID,
			Filename: filename,
			Title:    title,
			Path:     filename,
		}
		if err := store.CreateNote(note); err != nil {
			fmt.Printf("  ⚠ %s: created file but DB error: %v\n", filename, err)
		}

		fmt.Printf("  ✓ %s\n", filename)
		imported++
	}

	fmt.Printf("\n%d imported, %d skipped\n", imported, skipped)
	return nil
}

func extractTitle(props map[string]notionapi.Property) string {
	for _, p := range props {
		if p.Type == "title" && len(p.Title) > 0 && p.Title[0].Text != nil {
			return p.Title[0].Text.Content
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().Int("limit", 0, "Maximum number of pages to import")
}
