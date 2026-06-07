package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ishyverma/vault-sync/internal/config"
	notionapi "github.com/ishyverma/vault-sync/internal/connectors/notion"
	"github.com/ishyverma/vault-sync/internal/storage"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [backend]",
	Short: "Bulk-import notes from a backend",
	Long: `Bulk-imports existing notes from a connected backend into the local vault.

Supported backends: notion

Examples:
  vault import notion         Import all Notion pages from the target workspace
  vault import notion --limit 10`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend := args[0]

		switch backend {
		case "notion":
			return importFromNotion(cmd)
		default:
			return fmt.Errorf("unsupported backend: %s", backend)
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

		note := &storage.Note{
			ID:       page.ID,
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
